package middleware

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================
// Rate Limiter
// ============================================================

// RateLimiter - simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	times, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = []time.Time{now}
		return true
	}

	// Filter expired
	valid := []time.Time{}
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) < rl.limit {
		valid = append(valid, now)
		rl.visitors[ip] = valid
		return true
	}

	rl.visitors[ip] = valid
	return false
}

// Limit creates a Gin middleware from a RateLimiter
func Limit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.Allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// ============================================================
// Security Headers Middleware
// ============================================================

// SecurityHeaders adds security-related HTTP response headers.
// Configurable CSP to allow external CDN resources when needed.
func SecurityHeaders(cspAllow string) gin.HandlerFunc {
	// Default strict CSP - allow self + highlight.js CDN + data: images
	csp := "default-src 'self'; " +
		"script-src 'self' https://cdnjs.cloudflare.com; " +
		"style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; " +
		"img-src 'self' data: https: blob:; " +
		"media-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self';"

	if cspAllow != "" {
		csp = cspAllow
	}

	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", csp)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}

// ============================================================
// CSRF Middleware (Gin-compatible)
// ============================================================

// CSRFMiddleware validates Origin/Referer headers for state-changing requests.
// Should be applied to all POST/PUT/DELETE/PATCH routes.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" ||
			c.Request.Method == "DELETE" || c.Request.Method == "PATCH" {

			origin := c.Request.Header.Get("Origin")
			referer := c.Request.Header.Get("Referer")
			host := c.Request.Host

			// If both headers are missing, reject (except API clients with custom auth)
			if origin == "" && referer == "" {
				// Only allow if X-Requested-With header is present (AJAX requests)
				if c.Request.Header.Get("X-Requested-With") != "XMLHttpRequest" {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": "CSRF validation failed: missing origin",
					})
					return
				}
			}

			if origin != "" && !isSameOrigin(origin, host) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "CSRF validation failed: origin mismatch",
				})
				return
			}

			if referer != "" && !isSameOrigin(referer, host) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "CSRF validation failed: referer mismatch",
				})
				return
			}
		}
		c.Next()
	}
}

func isSameOrigin(originOrReferer, host string) bool {
	// Parse the origin/referer URL
	u, err := url.Parse(originOrReferer)
	if err != nil {
		return false
	}

	originHost := u.Hostname()

	// If no port in origin, allow matching by hostname only
	// If port present, require full host:port match
	if u.Port() == "" {
		return originHost == extractHost(host)
	}
	return u.Host == host
}

func extractHost(hostPort string) string {
	for i := len(hostPort) - 1; i >= 0; i-- {
		if hostPort[i] == ':' {
			return hostPort[:i]
		}
	}
	return hostPort
}
