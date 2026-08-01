package database

import (
	"log"
	"os"
	"path/filepath"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.Config) {
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create db directory: %v", err)
	}

	// upload directory
	if err := os.MkdirAll(filepath.Join(cfg.UploadDir, "images"), 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.UploadDir, "videos"), 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.UploadDir, "audios"), 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	DB = db
	migrate()
	seed(cfg)
}

func migrate() {
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Tag{},
		&models.Post{},
		&models.Section{},
		&models.Comment{},
		&models.Order{},
	); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
}

func seed(cfg *config.Config) {
	// Seed admin
	var count int64
	DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		hashed, _ := models.HashPassword(cfg.AdminPass)
		DB.Create(&models.User{
			Username: cfg.AdminUser,
			Password: hashed,
			Role:     "admin",
		})
	}

	// Default categories
	var catCount int64
	DB.Model(&models.Category{}).Count(&catCount)
	if catCount == 0 {
		DB.Create(&models.Category{Name: "随笔", Slug: "essay"})
		DB.Create(&models.Category{Name: "技术", Slug: "tech"})
		DB.Create(&models.Category{Name: "产品", Slug: "product"})
	}

	// Sample section
	var secCount int64
	DB.Model(&models.Section{}).Count(&secCount)
	if secCount == 0 {
		DB.Create(&models.Section{
			Name:        "深度专栏",
			Slug:        "deep-reads",
			Description: "高质量深度长文，值得细细品读",
			Price:       9.9,
		})
		DB.Create(&models.Section{
			Name:        "免费精选",
			Slug:        "free-picks",
			Description: "精选免费内容，与你分享知识的乐趣",
			Price:       0,
		})
	}

	// Sample posts
	var postCount int64
	DB.Model(&models.Post{}).Count(&postCount)
	if postCount == 0 {
		p1 := models.Post{
			Title:      "欢迎来到全新博客",
			Slug:       "welcome",
			Content:    "## 你好！\n\n这是我的全新博客。在这里，我会分享**技术笔记**、**生活感悟**和各种有趣的想法。\n\n![blog](https://images.unsplash.com/photo-1499750310107-5fef28a66643?w=800)\n\n### 这个博客使用什么技术？\n\n- **后端**：Go + Gin + GORM\n- **数据库**：SQLite\n- **前端**：原生 HTML/CSS/JS + Markdown 渲染\n\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n期待与你的交流！",
			CategoryID: 2,
			Status:     "published",
			Pinned:     true,
		}
		DB.Create(&p1)

		p2 := models.Post{
			Title:      "Go 语言并发编程之美",
			Slug:       "go-concurrency",
			Content:    "## Go 并发编程\n\nGo 语言的 goroutine 和 channel 让并发编程变得简单而优雅。\n\n### Goroutine\n\n```go\ngo func() {\n    fmt.Println(\"Hello from goroutine\")\n}()\n```\n\n### Channel\n\n```go\nch := make(chan int, 1)\nch <- 42\nfmt.Println(<-ch)\n```\n\nGo 的并发哲学是：**不要通过共享内存来通信，而要通过通信来共享内存。**",
			CategoryID: 2,
			Status:     "published",
		}
		DB.Create(&p2)

		p3 := models.Post{
			Title:      "设计系统的思考与实践",
			Slug:       "design-system",
			Content:    "## 设计系统\n\n> 好的设计系统不是规则的集合，而是价值观的表达。\n\n### 什么是设计系统？\n\n设计系统是一套可复用的**组件**、**模式**和**指南**，帮助团队构建一致的用户体验。\n\n### 核心原则\n\n1. **一致性** - 统一的设计语言\n2. **可复用** - 模块化的组件\n3. **可扩展** - 灵活的架构\n\n### 实践\n\n从按钮到页面布局，每一个元素都应该遵循设计系统的规范。",
			CategoryID: 3,
			Status:     "published",
		}
		DB.Create(&p3)

		// Associate first post with paid section
		var section models.Section
		DB.Where("slug = ?", "deep-reads").First(&section)
		DB.Model(&p1).Association("Sections").Append(&section)
	}
}
