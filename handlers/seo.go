package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/models"
)

// RobotsTxt serves /robots.txt
func RobotsTxt(c *gin.Context) {
	cfg := config.Load()
	sb := strings.Builder{}
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")
	sb.WriteString(fmt.Sprintf("Sitemap: %s/sitemap.xml\n", cfg.SiteURL))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sb.String()))
}

// SitemapXML serves /sitemap.xml
func SitemapXML(c *gin.Context) {
	cfg := config.Load()

	var posts []models.Post
	database.DB.Where("status = ?", "published").Order("created_at DESC").Find(&posts)

	var sections []models.Section
	database.DB.Where("is_active = ?", true).Find(&sections)

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	// Homepage
	sb.WriteString("<url>")
	sb.WriteString(fmt.Sprintf("<loc>%s/</loc>", cfg.SiteURL))
	sb.WriteString("<changefreq>daily</changefreq>")
	sb.WriteString("<priority>1.0</priority>")
	sb.WriteString("</url>")

	// About page
	sb.WriteString("<url>")
	sb.WriteString(fmt.Sprintf("<loc>%s/about</loc>", cfg.SiteURL))
	sb.WriteString("<changefreq>monthly</changefreq>")
	sb.WriteString("<priority>0.5</priority>")
	sb.WriteString("</url>")

	// Sections list
	sb.WriteString("<url>")
	sb.WriteString(fmt.Sprintf("<loc>%s/sections</loc>", cfg.SiteURL))
	sb.WriteString("<changefreq>weekly</changefreq>")
	sb.WriteString("<priority>0.7</priority>")
	sb.WriteString("</url>")

	// Section detail pages
	for _, s := range sections {
		sb.WriteString("<url>")
		sb.WriteString(fmt.Sprintf("<loc>%s/section/%s</loc>", cfg.SiteURL, s.Slug))
		sb.WriteString("<changefreq>weekly</changefreq>")
		sb.WriteString("<priority>0.6</priority>")
		sb.WriteString("</url>")
	}

	// Post pages
	for _, p := range posts {
		sb.WriteString("<url>")
		sb.WriteString(fmt.Sprintf("<loc>%s/post/%s</loc>", cfg.SiteURL, p.Slug))
		sb.WriteString("<changefreq>weekly</changefreq>")
		sb.WriteString("<priority>0.8</priority>")
		if p.UpdatedAt != "" && p.UpdatedAt != p.CreatedAt {
			sb.WriteString(fmt.Sprintf("<lastmod>%s</lastmod>", p.UpdatedAt))
		} else if p.CreatedAt != "" {
			sb.WriteString(fmt.Sprintf("<lastmod>%s</lastmod>", p.CreatedAt))
		}
		sb.WriteString("</url>")
	}

	sb.WriteString("</urlset>")

	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(sb.String()))
}
