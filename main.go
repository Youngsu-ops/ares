package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/handlers"
	"youngsu-blog-plus/middleware"
	"youngsu-blog-plus/utils"
)

func main() {
	cfg := config.Load()
	database.Init(cfg)

	// Gin mode from config (default: release)
	gin.SetMode(cfg.GinMode)
	r := gin.Default()

	// Limit multipart form memory (10MB max in memory, rest to temp file)
	r.MaxMultipartMemory = 10 << 20 // 10 MB

	// Trusted proxy for reverse proxy / CDN (ESA, Nginx, etc.)
	if cfg.TrustedProxy != "" {
		if err := r.SetTrustedProxies(strings.Split(cfg.TrustedProxy, ",")); err != nil {
			log.Printf("Warning: failed to set trusted proxies: %v", err)
		}
	}

	// Security Headers (applied to ALL responses)
	r.Use(middleware.SecurityHeaders(""))

	// Sessions with secure defaults
	sessionSecret := []byte(cfg.SessionSecret)
	store := cookie.NewStore(sessionSecret)
	store.Options(sessions.Options{
		MaxAge:   86400 * 7,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(cfg.SiteURL),
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("blogsession", store))

	// CSRF protection on all state-changing routes
	r.Use(middleware.CSRFMiddleware())

	// Rate limiter: 5 attempts per minute per IP for login/register
	loginLimiter := middleware.NewRateLimiter(5, time.Minute)
	registerLimiter := middleware.NewRateLimiter(3, time.Minute)

	// Static files
	r.Static("/static", "./static")

	// Load templates - combine root + subdirectory templates in a single set
	funcMap := template.FuncMap{
		"renderMarkdown": func(content string) template.HTML {
			return template.HTML(utils.RenderMarkdown(content))
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"range1": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i + 1
			}
			return result
		},
		"dateFmt": func(s string) string {
			if len(s) >= 10 {
				return s[:10]
			}
			return s
		},
		"formatPrice": func(p float64) string {
			if p <= 0 {
				return "免费"
			}
			return fmt.Sprintf("¥%.2f", p)
		},
		"formatDate": func(s string) string {
			if s == "" {
				return ""
			}
			if len(s) >= 19 {
				return s[:10] + " " + s[11:19]
			}
			if len(s) >= 10 {
				return s[:10]
			}
			return s
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i+1 < len(values); i += 2 {
				key := values[i].(string)
				dict[key] = values[i+1]
			}
			return dict
		},
		"printf": func(format string, a ...interface{}) string {
			return fmt.Sprintf(format, a...)
		},
		"slice": func(s string, start, end int) string {
			runes := []rune(s)
			if start < 0 || start > len(runes) {
				return ""
			}
			if end > len(runes) {
				end = len(runes)
			}
			return string(runes[start:end])
		},
		"adminPrefix": func() string {
			return "/" + cfg.AdminPath
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob(filepath.Join("templates", "*.html")))
	template.Must(tmpl.ParseGlob(filepath.Join("templates", "**", "*.html")))
	r.SetHTMLTemplate(tmpl)

	// ===== SEO Routes =====
	r.GET("/robots.txt", handlers.RobotsTxt)
	r.GET("/sitemap.xml", handlers.SitemapXML)

	// ===== Public Routes =====
	r.GET("/", handlers.Timeline)
	r.GET("/post/:slug", middleware.SectionAccessControl(), handlers.ShowPost)
	r.GET("/category/:slug", handlers.ShowCategory)
	r.GET("/search", handlers.Search)
	r.GET("/about", handlers.About)
	r.GET("/feed", handlers.RSSFeed)

	// Sections
	r.GET("/sections", handlers.AllSections)
	r.GET("/section/:slug", handlers.ShowSection)

	// ===== Authentication Routes (rate-limited) =====
	r.GET("/auth/login", handlers.ShowLogin)
	r.POST("/auth/login", middleware.Limit(loginLimiter), handlers.Login)
	r.GET("/auth/register", handlers.ShowRegister)
	r.POST("/auth/register", middleware.Limit(registerLimiter), handlers.Register)
	r.GET("/auth/logout", handlers.Logout)

	// ===== Authenticated Routes =====
	auth := r.Group("", middleware.AuthRequired())
	{
		auth.POST("/post/:slug/comment", handlers.CreateComment)
		auth.GET("/section/:slug/purchase", handlers.PurchaseSection)
		auth.POST("/section/:slug/purchase", handlers.ConfirmPurchase)
	}

	// ===== Admin Routes =====
	adminPath := "/" + cfg.AdminPath
	admin := r.Group(adminPath, middleware.AdminRequired())
	{
		admin.GET("/dashboard", handlers.Dashboard)

		// Posts
		admin.GET("/posts", handlers.AdminPostList)
		admin.GET("/posts/new", handlers.AdminPostCreate)
		admin.POST("/posts/new", handlers.AdminPostStore)
		admin.GET("/posts/:id/edit", handlers.AdminPostEdit)
		admin.POST("/posts/:id/edit", handlers.AdminPostUpdate)
		admin.DELETE("/posts/:id", handlers.AdminPostDelete)

		// Upload
		admin.POST("/upload", handlers.UploadFile)

		// Categories
		admin.GET("/categories", handlers.AdminCategoryList)
		admin.POST("/categories", handlers.AdminCategoryStore)
		admin.DELETE("/categories/:id", handlers.AdminCategoryDelete)

		// Sections
		admin.GET("/sections", handlers.AdminSectionList)
		admin.GET("/sections/new", handlers.AdminSectionCreate)
		admin.POST("/sections/new", handlers.AdminSectionStore)
		admin.GET("/sections/:id/edit", handlers.AdminSectionEdit)
		admin.POST("/sections/:id/edit", handlers.AdminSectionUpdate)
		admin.DELETE("/sections/:id", handlers.AdminSectionDelete)
		admin.GET("/sections/:id/posts", handlers.AdminSectionManagePosts)
		admin.POST("/sections/:id/posts", handlers.AdminSectionTogglePost)

		// Comments
		admin.GET("/comments", handlers.AdminCommentList)
		admin.POST("/comments/:id/approve", handlers.AdminCommentApprove)
		admin.POST("/comments/:id/reject", handlers.AdminCommentReject)
		admin.DELETE("/comments/:id", handlers.AdminCommentDelete)
	}

	// ===== 404 =====
	r.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", gin.H{
			"SiteName":     cfg.SiteName,
			"SiteDesc":     cfg.SiteDesc,
			"SiteURL":      cfg.SiteURL,
			"SiteKeywords": cfg.SiteKeywords,
			"OGImage":      cfg.OGImage,
			"SeoTitle":     "404 - " + cfg.SiteName,
			"SeoURL":       cfg.SiteURL + "/",
		})
	})

	log.Println("============================================")
	log.Printf("  %s 启动成功!", cfg.SiteName)
	log.Printf("  访问地址: http://localhost:%s", cfg.Port)
	log.Printf("  后台管理: http://localhost:%s/%s/dashboard", cfg.Port, cfg.AdminPath)
	log.Printf("  运行模式: %s", cfg.GinMode)
	if os.Getenv("BLOG_ADMIN_PASS") == "" {
		log.Printf("  管理员密码(自动生成): %s", cfg.AdminPass)
		log.Println("  [提示] 请通过 BLOG_ADMIN_PASS 环境变量设置密码")
	} else {
		log.Println("  管理员密码: [已通过环境变量设置]")
	}
	log.Println("============================================")

	if err := r.Run("0.0.0.0:" + cfg.Port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

// isHTTPS checks if site URL uses HTTPS scheme
func isHTTPS(siteURL string) bool {
	return strings.HasPrefix(strings.ToLower(siteURL), "https://")
}

