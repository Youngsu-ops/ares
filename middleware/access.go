package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/database"
	"youngsu-blog-plus/models"
)

// SectionAccessControl checks if user can access a post's paid sections
func SectionAccessControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)

		// If admin, allow all
		if user != nil && user.IsAdmin() {
			c.Next()
			return
		}

		// Check if post belongs to any paid sections
		postSlug := c.Param("slug")
		var post models.Post
		if err := database.DB.Preload("Sections").Where("slug = ?", postSlug).First(&post).Error; err != nil {
			c.Next()
			return
		}

		paidSections := GetPaidSections(post.Sections)
		if len(paidSections) == 0 {
			c.Next()
			return
		}

		if user == nil {
			// Not logged in - redirect to purchase page
			c.HTML(http.StatusOK, "purchase.html", gin.H{
				"Sections": paidSections,
				"Post":     post,
				"RequireAuth": true,
			})
			c.Abort()
			return
		}

		// Check if user has purchased
		var orders []models.Order
		sectionIDs := make([]uint, len(paidSections))
		for i, s := range paidSections {
			sectionIDs[i] = s.ID
		}
		database.DB.Where("user_id = ? AND section_id IN ? AND status = ?", user.ID, sectionIDs, "paid").Find(&orders)

		if len(orders) == 0 {
			c.HTML(http.StatusOK, "purchase.html", gin.H{
				"Sections": paidSections,
				"Post":     post,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetPaidSections returns sections with price > 0
func GetPaidSections(sections []models.Section) []models.Section {
	var paid []models.Section
	for _, s := range sections {
		if s.Price > 0 {
			paid = append(paid, s)
		}
	}
	return paid
}

// GetUserPurchasedSectionIDs returns set of purchased section IDs
func GetUserPurchasedSectionIDs(userID uint) map[uint]bool {
	result := make(map[uint]bool)
	var orders []models.Order
	database.DB.Where("user_id = ? AND status = ?", userID, "paid").Find(&orders)
	for _, o := range orders {
		result[o.SectionID] = true
	}
	return result
}
