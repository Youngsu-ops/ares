package handlers

import (
	"fmt"

	"youngsu-blog-plus/config"
)

// SeoData returns common SEO/template data used across all pages
func SeoData(overrides map[string]interface{}) map[string]interface{} {
	cfg := config.Load()
	data := map[string]interface{}{
		"SiteName":     cfg.SiteName,
		"SiteDesc":     cfg.SiteDesc,
		"SiteAuthor":   cfg.SiteAuthor,
		"SiteURL":      cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":      cfg.OGImage,
	}
	for k, v := range overrides {
		data[k] = v
	}
	return data
}

// AdminPath returns the full path for an admin route (e.g. "/admin/posts")
func AdminPath(suffix string) string {
	return fmt.Sprintf("/%s%s", config.Load().AdminPath, suffix)
}
