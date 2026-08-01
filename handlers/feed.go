package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/models"
)

func RSSFeed(c *gin.Context) {
	var posts []models.Post
	database.DB.Preload("Category").
		Where("status = ?", "published").
		Order("created_at DESC").
		Limit(20).
		Find(&posts)

	cfg := config.Load()

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<rss version="2.0"><channel>`)
	sb.WriteString(fmt.Sprintf("<title>%s</title>", cfg.SiteName))
	sb.WriteString(fmt.Sprintf("<description>%s</description>", cfg.SiteDesc))
	sb.WriteString(fmt.Sprintf("<link>%s/</link>", cfg.SiteURL))

	for _, post := range posts {
		sb.WriteString("<item>")
		sb.WriteString(fmt.Sprintf("<title>%s</title>", post.Title))
		sb.WriteString(fmt.Sprintf("<link>%s/post/%s</link>", cfg.SiteURL, post.Slug))
		sb.WriteString(fmt.Sprintf("<guid>%s/post/%s</guid>", cfg.SiteURL, post.Slug))
		sb.WriteString(fmt.Sprintf("<description>%s</description>", post.Excerpt))
		if post.CreatedAt != "" {
			t, err := time.Parse("2006-01-02 15:04:05", post.CreatedAt)
			if err == nil {
				sb.WriteString(fmt.Sprintf("<pubDate>%s</pubDate>", t.Format(time.RFC1123Z)))
			}
		}
		sb.WriteString("</item>")
	}

	sb.WriteString("</channel></rss>")

	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", []byte(sb.String()))
}
