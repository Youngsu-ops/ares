package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

type Config struct {
	Port          string
	DBPath        string
	SiteName      string
	SiteDesc      string
	SiteAuthor    string
	SiteURL       string
	SiteKeywords  string
	OGImage       string
	AdminUser     string
	AdminPass     string
	AdminPath     string
	UploadDir     string
	MaxUploadMB   int64
	SessionSecret string
	GinMode       string
	TrustedProxy  string
}

func Load() *Config {
	cfg := &Config{
		Port:          getEnv("BLOG_PORT", "8888"),
		DBPath:        getEnv("BLOG_DB", "data/blog.db"),
		SiteName:      getEnv("BLOG_NAME", "Youngsu's Blog"),
		SiteDesc:      "记录技术与生活的点滴，探索无限可能",
		SiteAuthor:    "Youngsu",
		SiteURL:       getEnv("BLOG_URL", "http://localhost:8888"),
		SiteKeywords:  getEnv("BLOG_KEYWORDS", "技术,编程,Go,Golang,博客,个人博客"),
		OGImage:       getEnv("BLOG_OG_IMAGE", "/static/img/og-default.png"),
		AdminUser:     getEnv("BLOG_ADMIN_USER", "admin"),
		AdminPass:     os.Getenv("BLOG_ADMIN_PASS"), // No default - must be set for production
		AdminPath:     getEnv("BLOG_ADMIN_PATH", "admin"),
		UploadDir:     getEnv("BLOG_UPLOAD", "static/uploads"),
		MaxUploadMB:   10,
		SessionSecret: getEnv("SESSION_SECRET", ""),
		GinMode:       getEnv("GIN_MODE", "release"),
		TrustedProxy:  getEnv("TRUSTED_PROXY", ""),
	}

	// Generate random session secret if not set
	if cfg.SessionSecret == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			// Fallback: use timestamp-based pseudo-random
			secret = []byte(fmt.Sprintf("youngsu-blog-fallback-%d", os.Getpid()))
		}
		cfg.SessionSecret = hex.EncodeToString(secret)
	}

	// Generate random admin password if not set
	if cfg.AdminPass == "" {
		pwd := make([]byte, 12)
		if _, err := rand.Read(pwd); err != nil {
			cfg.AdminPass = "changeme-please"
		} else {
			cfg.AdminPass = hex.EncodeToString(pwd)
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
