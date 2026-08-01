package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/middleware"
	"youngsu-blog-plus/models"
)

// ShowSection displays section details and its posts
func ShowSection(c *gin.Context) {
	slug := c.Param("slug")
	var section models.Section
	if err := database.DB.Where("slug = ?", slug).First(&section).Error; err != nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"SiteName": config.Load().SiteName,
		})
		return
	}

	var posts []models.Post
	database.DB.Model(&section).Preload("Category").
		Where("status = ?", "published").
		Association("Posts").Find(&posts)

	// Check if user has bought this section
	user := middleware.GetCurrentUser(c)
	hasAccess := true // free sections
	if section.Price > 0 {
		if user == nil {
			hasAccess = false
		} else {
			var count int64
			database.DB.Model(&models.Order{}).
				Where("user_id = ? AND section_id = ? AND status = ?", user.ID, section.ID, "paid").
				Count(&count)
			hasAccess = user.IsAdmin() || count > 0
		}
	}

	var categories []models.Category
	database.DB.Find(&categories)

	cfg := config.Load()
	c.HTML(http.StatusOK, "detail.html", gin.H{
		"SiteName":     cfg.SiteName,
		"SiteDesc":     cfg.SiteDesc,
		"SiteURL":      cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":      cfg.OGImage,
		"Section":      section,
		"Posts":        posts,
		"Categories":   categories,
		"CurrentUser":  user,
		"HasAccess":    hasAccess,
		"SeoTitle":     section.Name + " - " + cfg.SiteName,
		"SeoDesc":      section.Description,
		"SeoURL":       cfg.SiteURL + "/section/" + section.Slug,
	})
}

// PurchaseSection shows purchase form
func PurchaseSection(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	slug := c.Param("slug")
	var section models.Section
	if err := database.DB.Where("slug = ?", slug).First(&section).Error; err != nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"SiteName": config.Load().SiteName,
		})
		return
	}

	if section.Price <= 0 {
		c.Redirect(http.StatusFound, "/section/"+slug)
		return
	}

	var categories []models.Category
	database.DB.Find(&categories)

	cfg := config.Load()
	c.HTML(http.StatusOK, "purchase_standalone.html", gin.H{
		"SiteName":     cfg.SiteName,
		"SiteDesc":     cfg.SiteDesc,
		"SiteURL":      cfg.SiteURL,
		"SiteKeywords": cfg.SiteKeywords,
		"OGImage":      cfg.OGImage,
		"Section":      section,
		"Categories":   categories,
		"CurrentUser":  user,
		"SeoTitle":     "购买 " + section.Name + " - " + cfg.SiteName,
		"SeoDesc":      section.Description,
		"SeoURL":       cfg.SiteURL + "/section/" + section.Slug + "/purchase",
	})
}

// ConfirmPurchase processes the purchase (mock payment)
func ConfirmPurchase(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	slug := c.Param("slug")
	var section models.Section
	if err := database.DB.Where("slug = ?", slug).First(&section).Error; err != nil {
		c.Redirect(http.StatusFound, "/sections")
		return
	}

	// Check if already purchased
	var count int64
	database.DB.Model(&models.Order{}).
		Where("user_id = ? AND section_id = ? AND status = ?", user.ID, section.ID, "paid").
		Count(&count)
	if count > 0 {
		c.Redirect(http.StatusFound, "/section/"+slug)
		return
	}

	// Mock payment - in real app, integrate with payment gateway
	order := models.Order{
		UserID:    user.ID,
		SectionID: section.ID,
		Amount:    section.Price,
		Status:    "paid", // auto-confirm for demo
	}
	database.DB.Create(&order)

	c.Redirect(http.StatusFound, "/section/"+slug+"?purchased=1")
}

// AllSections lists all sections
func AllSections(c *gin.Context) {
	var sections []models.Section
	database.DB.Preload("Posts").Find(&sections)

	var categories []models.Category
	database.DB.Find(&categories)

	user := middleware.GetCurrentUser(c)
	var purchasedIDs map[uint]bool
	if user != nil {
		purchasedIDs = middleware.GetUserPurchasedSectionIDs(user.ID)
	} else {
		purchasedIDs = make(map[uint]bool)
	}

	cfg := config.Load()
	c.HTML(http.StatusOK, "list.html", gin.H{
		"SiteName":      cfg.SiteName,
		"SiteDesc":      cfg.SiteDesc,
		"SiteURL":       cfg.SiteURL,
		"SiteKeywords":  cfg.SiteKeywords,
		"OGImage":       cfg.OGImage,
		"Sections":      sections,
		"Categories":    categories,
		"PurchasedIDs":  purchasedIDs,
		"CurrentUser":   user,
		"SeoTitle":      "专栏 - " + cfg.SiteName,
		"SeoDesc":       "浏览 " + cfg.SiteName + " 的付费和免费专栏",
		"SeoURL":        cfg.SiteURL + "/sections",
	})
}

// AdminSectionList - manage sections in admin
func AdminSectionList(c *gin.Context) {
	var sections []models.Section
	database.DB.Find(&sections)
	c.HTML(http.StatusOK, "section_list.html", gin.H{
		"SiteName": config.Load().SiteName,
		"Sections": sections,
	})
}

func AdminSectionCreate(c *gin.Context) {
	c.HTML(http.StatusOK, "section_form.html", gin.H{
		"SiteName": config.Load().SiteName,
		"IsEdit":   false,
	})
}

func AdminSectionStore(c *gin.Context) {
	name := c.PostForm("name")
	slug := c.PostForm("slug")
	desc := c.PostForm("description")
	coverImage := c.PostForm("cover_image")
	price, _ := parseFloat(c.PostForm("price"))

	section := models.Section{
		Name:        name,
		Slug:        slug,
		Description: desc,
		CoverImage:  coverImage,
		Price:       price,
	}
	database.DB.Create(&section)
	c.Redirect(http.StatusFound, AdminPath("/sections"))
}

func AdminSectionEdit(c *gin.Context) {
	id := c.Param("id")
	var section models.Section
	if err := database.DB.First(&section, id).Error; err != nil {
		c.Redirect(http.StatusFound, AdminPath("/sections"))
		return
	}

	c.HTML(http.StatusOK, "section_form.html", gin.H{
		"SiteName": config.Load().SiteName,
		"IsEdit":   true,
		"Section":  section,
	})
}

func AdminSectionUpdate(c *gin.Context) {
	id := c.Param("id")
	var section models.Section
	if err := database.DB.First(&section, id).Error; err != nil {
		c.Redirect(http.StatusFound, AdminPath("/sections"))
		return
	}

	section.Name = c.PostForm("name")
	section.Slug = c.PostForm("slug")
	section.Description = c.PostForm("description")
	section.CoverImage = c.PostForm("cover_image")
	price, _ := parseFloat(c.PostForm("price"))
	section.Price = price

	database.DB.Save(&section)
	c.Redirect(http.StatusFound, AdminPath("/sections"))
}

func AdminSectionDelete(c *gin.Context) {
	id := c.Param("id")
	database.DB.Delete(&models.Section{}, id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminSectionManagePosts - manage which posts belong to a section
func AdminSectionManagePosts(c *gin.Context) {
	id := c.Param("id")
	var section models.Section
	if err := database.DB.First(&section, id).Error; err != nil {
		c.Redirect(http.StatusFound, AdminPath("/sections"))
		return
	}

	var allPosts []models.Post
	database.DB.Preload("Category").Where("status = ?", "published").Order("created_at DESC").Find(&allPosts)

	var sectionPosts []models.Post
	database.DB.Model(&section).Association("Posts").Find(&sectionPosts)

	sectionPostIDs := make(map[uint]bool)
	for _, p := range sectionPosts {
		sectionPostIDs[p.ID] = true
	}

	c.HTML(http.StatusOK, "section_posts.html", gin.H{
		"SiteName":       config.Load().SiteName,
		"Section":        section,
		"AllPosts":       allPosts,
		"SectionPostIDs": sectionPostIDs,
	})
}

func AdminSectionTogglePost(c *gin.Context) {
	sectionID := c.Param("id")
	postID := c.PostForm("post_id")

	var section models.Section
	var post models.Post
	database.DB.First(&section, sectionID)
	database.DB.First(&post, postID)

	var count int64
	database.DB.Table("section_posts").
		Where("section_id = ? AND post_id = ?", section.ID, post.ID).
		Count(&count)

	if count > 0 {
		database.DB.Model(&section).Association("Posts").Delete(&post)
	} else {
		database.DB.Model(&section).Association("Posts").Append(&post)
	}

	c.Redirect(http.StatusFound,
		AdminPath(fmt.Sprintf("/sections/%s/posts", sectionID)))
}

func parseFloat(s string) (float64, error) {
	f := 0.0
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
