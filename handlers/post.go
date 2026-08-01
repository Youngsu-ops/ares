package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/middleware"
	"youngsu-blog-plus/models"
	"youngsu-blog-plus/utils"
)

// Timeline is the futuristic homepage with chronological post tree
func Timeline(c *gin.Context) {
	var posts []models.Post
	database.DB.Preload("Category").Preload("Tags").Preload("Sections").
		Where("status = ?", "published").
		Order("created_at DESC").
		Find(&posts)

	for i := range posts {
		if posts[i].Excerpt == "" {
			posts[i].Excerpt = utils.ExcerptFromContent(posts[i].Content, 120)
		}
	}

	var sections []models.Section
	database.DB.Find(&sections)

	var categories []models.Category
	database.DB.Find(&categories)

	user := middleware.GetCurrentUser(c)
	sectionIDs := make([]uint, 0)
	if user != nil {
		sectionIDs = make([]uint, 0)
		_ = sectionIDs
	}

	cfg := config.Load()
	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"Posts":      posts,
		"Categories": categories,
		"Sections":   sections,
		"SiteName":   cfg.SiteName,
		"SiteDesc":   cfg.SiteDesc,
		"SiteURL":    cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":    cfg.OGImage,
		"SeoTitle":   cfg.SiteName + " - " + cfg.SiteDesc,
		"SeoDesc":    cfg.SiteDesc,
		"SeoURL":     cfg.SiteURL + "/",
		"CurrentUser": user,
	})
}

// Index is the legacy list view (redirect to timeline)
func Index(c *gin.Context) {
	Timeline(c)
}

// ShowPost displays a single post
func ShowPost(c *gin.Context) {
	slug := c.Param("slug")

	var post models.Post
	if err := database.DB.Preload("Category").Preload("Tags").Preload("Sections").
		Where("slug = ? AND status = ?", slug, "published").
		First(&post).Error; err != nil {
		cfg := config.Load()
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"SiteName":     cfg.SiteName,
			"SiteDesc":     cfg.SiteDesc,
			"SiteURL":      cfg.SiteURL,
			"SiteKeywords": cfg.SiteKeywords,
			"OGImage":      cfg.OGImage,
			"SeoTitle":     "404 - " + cfg.SiteName,
			"SeoURL":       cfg.SiteURL + "/",
		})
		return
	}

	// Check paid section access
	cfg := config.Load()
	user := middleware.GetCurrentUser(c)
	if user != nil && !user.IsAdmin() {
		paidSections := middleware.GetPaidSections(post.Sections)
		if len(paidSections) > 0 {
			purchasedIDs := middleware.GetUserPurchasedSectionIDs(user.ID)
			for _, s := range paidSections {
				if !purchasedIDs[s.ID] {
					c.HTML(http.StatusOK, "purchase.html", gin.H{
						"SiteName":     cfg.SiteName,
						"SiteDesc":     cfg.SiteDesc,
						"SiteURL":      cfg.SiteURL,
						"SiteKeywords": cfg.SiteKeywords,
						"OGImage":      cfg.OGImage,
						"Sections":     paidSections,
						"Post":         post,
						"SeoTitle":     "付费阅读 - " + cfg.SiteName,
						"SeoURL":       cfg.SiteURL + "/post/" + post.Slug,
					})
					return
				}
			}
		}
	} else if user == nil {
		paidSections := middleware.GetPaidSections(post.Sections)
		if len(paidSections) > 0 {
			c.HTML(http.StatusOK, "purchase.html", gin.H{
				"SiteName":     cfg.SiteName,
				"SiteDesc":     cfg.SiteDesc,
				"SiteURL":      cfg.SiteURL,
				"SiteKeywords": cfg.SiteKeywords,
				"OGImage":      cfg.OGImage,
				"Sections":     paidSections,
				"Post":         post,
				"RequireAuth":  true,
				"SeoTitle":     "付费阅读 - " + cfg.SiteName,
				"SeoURL":       cfg.SiteURL + "/post/" + post.Slug,
			})
			return
		}
	}

	database.DB.Model(&post).UpdateColumn("views", post.Views+1)
	htmlContent := template.HTML(utils.RenderMarkdown(post.Content))

	var comments []models.Comment
	database.DB.Preload("User").
		Where("post_id = ? AND status = ?", post.ID, "approved").
		Order("created_at DESC").Find(&comments)

	var categories []models.Category
	database.DB.Find(&categories)

	var sections []models.Section
	database.DB.Find(&sections)

	cfg = config.Load()
	seoDesc := post.Excerpt
	if seoDesc == "" {
		seoDesc = utils.ExcerptFromContent(post.Content, 160)
	}
	seoImage := cfg.OGImage
	if post.CoverImage != "" {
		seoImage = cfg.SiteURL + "/" + post.CoverImage
	}

	c.HTML(http.StatusOK, "post.html", gin.H{
		"Post":        post,
		"HTMLContent": htmlContent,
		"Comments":    comments,
		"Categories":  categories,
		"Sections":    sections,
		"SiteName":    cfg.SiteName,
		"SiteDesc":    cfg.SiteDesc,
		"SiteURL":     cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":     cfg.OGImage,
		"CurrentUser": user,
		"SeoTitle":    post.Title + " - " + cfg.SiteName,
		"SeoDesc":     seoDesc,
		"SeoURL":      cfg.SiteURL + "/post/" + post.Slug,
		"SeoImage":    seoImage,
		"SeoType":     "article",
		"SeoPublishedAt": post.CreatedAt,
		"SeoModifiedAt":  post.UpdatedAt,
	})
}

func ShowCategory(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := 10

	var category models.Category
	if err := database.DB.Where("slug = ?", slug).First(&category).Error; err != nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"SiteName": config.Load().SiteName,
		})
		return
	}

	var posts []models.Post
	var total int64
	database.DB.Model(&models.Post{}).
		Where("category_id = ? AND status = ?", category.ID, "published").
		Count(&total)
	database.DB.Preload("Category").Preload("Tags").Preload("Sections").
		Where("category_id = ? AND status = ?", category.ID, "published").
		Order("created_at DESC").
		Offset((page - 1) * perPage).Limit(perPage).
		Find(&posts)

	for i := range posts {
		if posts[i].Excerpt == "" {
			posts[i].Excerpt = utils.ExcerptFromContent(posts[i].Content, 120)
		}
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	var categories []models.Category
	database.DB.Find(&categories)

	var sections []models.Section
	database.DB.Find(&sections)

	cfg := config.Load()
	c.HTML(http.StatusOK, "category.html", gin.H{
		"Posts":        posts,
		"Categories":   categories,
		"Sections":     sections,
		"CategoryName": category.Name,
		"SiteName":     cfg.SiteName,
		"SiteDesc":     cfg.SiteDesc,
		"SiteURL":      cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":      cfg.OGImage,
		"SeoTitle":     category.Name + " - " + cfg.SiteName,
		"SeoDesc":      "浏览 " + category.Name + " 分类下的所有文章",
		"SeoURL":       cfg.SiteURL + "/category/" + category.Slug,
		"CurrentPage":  page,
		"TotalPages":   totalPages,
		"CurrentUser":  middleware.GetCurrentUser(c),
	})
}

func Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var posts []models.Post
	database.DB.Preload("Category").Preload("Tags").Preload("Sections").
		Where("status = ? AND (title LIKE ? OR content LIKE ?)", "published", "%"+q+"%", "%"+q+"%").
		Order("created_at DESC").
		Find(&posts)

	for i := range posts {
		if posts[i].Excerpt == "" {
			posts[i].Excerpt = utils.ExcerptFromContent(posts[i].Content, 120)
		}
	}

	var categories []models.Category
	database.DB.Find(&categories)

	var sections []models.Section
	database.DB.Find(&sections)

	cfg := config.Load()
	c.HTML(http.StatusOK, "category.html", gin.H{
		"Posts":       posts,
		"Categories":  categories,
		"Sections":    sections,
		"SearchQuery": q,
		"SiteName":    cfg.SiteName,
		"SiteDesc":    cfg.SiteDesc,
		"SiteURL":     cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":     cfg.OGImage,
		"SeoTitle":    "搜索: " + q + " - " + cfg.SiteName,
		"SeoDesc":     "搜索关键词 '" + q + "' 的结果",
		"SeoURL":      cfg.SiteURL + "/search?q=" + q,
		"CurrentPage": 1,
		"TotalPages":  1,
		"CurrentUser": middleware.GetCurrentUser(c),
	})
}

func About(c *gin.Context) {
	var categories []models.Category
	database.DB.Find(&categories)
	var sections []models.Section
	database.DB.Find(&sections)

	cfg := config.Load()
	c.HTML(http.StatusOK, "about.html", gin.H{
		"SiteName":     cfg.SiteName,
		"SiteAuthor":   cfg.SiteAuthor,
		"SiteDesc":     cfg.SiteDesc,
		"SiteURL":      cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":      cfg.OGImage,
		"Categories":   categories,
		"Sections":     sections,
		"CurrentUser":  middleware.GetCurrentUser(c),
		"SeoTitle":     "关于 - " + cfg.SiteName,
		"SeoDesc":      cfg.SiteDesc,
		"SeoURL":       cfg.SiteURL + "/about",
	})
}

// CreateComment handles comment submission (requires login)
func CreateComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	slug := c.Param("slug")
	var post models.Post
	if err := database.DB.Where("slug = ?", slug).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}

	content := c.PostForm("content")
	if content == "" {
		c.Redirect(http.StatusFound, "/post/"+slug+"#comments")
		return
	}

	// Sanitize using bluemonday (defense-in-depth)
	content = utils.SanitizeHTML(content)

	comment := models.Comment{
		PostID:  post.ID,
		UserID:  &user.ID,
		Author:  user.Username,
		Content: content,
		Status:  "pending",
	}
	database.DB.Create(&comment)

	c.Redirect(http.StatusFound, "/post/"+slug+"#comments")
}
