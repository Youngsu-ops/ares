package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/database"
	"youngsu-blog-plus/models"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID == nil {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		c.Set("currentUser", &user)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID == nil {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil || user.Role != "admin" {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		c.Set("currentUser", &user)
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) *models.User {
	u, exists := c.Get("currentUser")
	if !exists {
		// Try from session as fallback
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID != nil {
			var user models.User
			if err := database.DB.First(&user, userID).Error; err == nil {
				c.Set("currentUser", &user)
				return &user
			}
		}
		return nil
	}
	return u.(*models.User)
}

func IsAuthenticated(c *gin.Context) bool {
	return GetCurrentUser(c) != nil
}
