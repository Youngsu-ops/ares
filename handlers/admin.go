package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/models"
	"youngsu-blog-plus/utils"
)

func Dashboard(c *gin.Context) {
	var postCount, commentCount, sectionCount, userCount, orderCount int64
	database.DB.Model(&models.Post{}).Count(&postCount)
	database.DB.Model(&models.Comment{}).Count(&commentCount)
	database.DB.Model(&models.Section{}).Count(&sectionCount)
	database.DB.Model(&models.User{}).Count(&userCount)
	database.DB.Model(&models.Order{}).Where("status = ?", "paid").Count(&orderCount)

	var pendingComments int64
	database.DB.Model(&models.Comment{}).Where("status = ?", "pending").Count(&pendingComments)

	var totalRevenue float64
	database.DB.Model(&models.Order{}).Where("status = ?", "paid").
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&totalRevenue)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"SiteName":        config.Load().SiteName,
		"PostCount":       postCount,
		"CommentCount":    commentCount,
		"SectionCount":    sectionCount,
		"UserCount":       userCount,
		"OrderCount":      orderCount,
		"PendingComments": pendingComments,
		"TotalRevenue":    totalRevenue,
	})
}

func AdminPostList(c *gin.Context) {
	var posts []models.Post
	database.DB.Preload("Category").Order("created_at DESC").Find(&posts)
	c.HTML(http.StatusOK, "post_list.html", gin.H{
		"SiteName": config.Load().SiteName,
		"Posts":    posts,
	})
}

func AdminPostCreate(c *gin.Context) {
	var categories []models.Category
	database.DB.Find(&categories)
	var sections []models.Section
	database.DB.Find(&sections)

	c.HTML(http.StatusOK, "post_form.html", gin.H{
		"SiteName":       config.Load().SiteName,
		"Categories":     categories,
		"Sections":       sections,
		"PostSectionIDs": make(map[uint]bool),
		"IsEdit":         false,
	})
}

func AdminPostStore(c *gin.Context) {
	title := c.PostForm("title")
	content := c.PostForm("content")
	categoryID, _ := strconv.Atoi(c.PostForm("category_id"))
	status := c.PostForm("status")
	tagNames := c.PostForm("tags")
	excerpt := c.PostForm("excerpt")
	coverImage := c.PostForm("cover_image")
	audioURL := c.PostForm("audio_url")
	videoURL := c.PostForm("video_url")
	sectionIDs := c.PostFormArray("section_ids")

	slug := c.PostForm("slug")
	if slug == "" {
		slug = utils.Slugify(title)
	}

	if status == "" {
		status = "draft"
	}

	// Sanitize content
	content = strings.TrimSpace(content)

	// Auto-generate excerpt if not provided
	if excerpt == "" {
		excerpt = utils.ExcerptFromContent(content, 150)
	}

	post := models.Post{
		Title:      title,
		Slug:       slug,
		Content:    content,
		Excerpt:    excerpt,
		CoverImage: coverImage,
		AudioURL:   audioURL,
		VideoURL:   videoURL,
		CategoryID: uint(categoryID),
		Status:     status,
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := database.DB.Create(&post).Error; err != nil {
		var categories []models.Category
		database.DB.Find(&categories)
		c.HTML(http.StatusOK, "post_form.html", gin.H{
			"SiteName":   config.Load().SiteName,
			"Categories": categories,
			"IsEdit":     false,
			"Error":      "保存失败，Slug可能已存在",
			"Post":       post,
		})
		return
	}

	// Handle sections
	for _, sid := range sectionIDs {
		id, err := strconv.Atoi(sid)
		if err == nil {
			var section models.Section
			database.DB.First(&section, uint(id))
			database.DB.Model(&post).Association("Sections").Append(&section)
		}
	}

	// Handle tags
	if tagNames != "" {
		tags := strings.Split(tagNames, ",")
		for _, tagName := range tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagSlug := utils.Slugify(tagName)
			var tag models.Tag
			result := database.DB.Where("slug = ?", tagSlug).First(&tag)
			if result.Error != nil {
				tag = models.Tag{Name: tagName, Slug: tagSlug}
				database.DB.Create(&tag)
			}
			database.DB.Model(&post).Association("Tags").Append(&tag)
		}
	}

	c.Redirect(http.StatusFound, AdminPath("/posts"))
}

func AdminPostEdit(c *gin.Context) {
	id := c.Param("id")
	var post models.Post
	if err := database.DB.Preload("Tags").Preload("Sections").First(&post, id).Error; err != nil {
		c.Redirect(http.StatusFound, AdminPath("/posts"))
		return
	}

	var categories []models.Category
	database.DB.Find(&categories)
	var sections []models.Section
	database.DB.Find(&sections)

	tagNames := ""
	for i, tag := range post.Tags {
		if i > 0 {
			tagNames += ", "
		}
		tagNames += tag.Name
	}

	// Get post's section IDs
	postSectionIDs := make(map[uint]bool)
	for _, s := range post.Sections {
		postSectionIDs[s.ID] = true
	}

	c.HTML(http.StatusOK, "post_form.html", gin.H{
		"SiteName":       config.Load().SiteName,
		"Categories":     categories,
		"Sections":       sections,
		"Post":           post,
		"TagNames":       tagNames,
		"PostSectionIDs": postSectionIDs,
		"IsEdit":         true,
	})
}

func AdminPostUpdate(c *gin.Context) {
	id := c.Param("id")
	var post models.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.Redirect(http.StatusFound, AdminPath("/posts"))
		return
	}

	post.Title = c.PostForm("title")
	post.Content = strings.TrimSpace(c.PostForm("content"))
	post.Excerpt = c.PostForm("excerpt")
	// Auto-generate excerpt if not provided
	if post.Excerpt == "" {
		post.Excerpt = utils.ExcerptFromContent(post.Content, 150)
	}
	post.CoverImage = c.PostForm("cover_image")
	post.AudioURL = c.PostForm("audio_url")
	post.VideoURL = c.PostForm("video_url")
	post.Status = c.PostForm("status")
	categoryID, _ := strconv.Atoi(c.PostForm("category_id"))
	post.CategoryID = uint(categoryID)

	slug := c.PostForm("slug")
	if slug != "" {
		post.Slug = slug
	}

	database.DB.Save(&post)

	// Update sections
	database.DB.Model(&post).Association("Sections").Clear()
	sectionIDs := c.PostFormArray("section_ids")
	for _, sid := range sectionIDs {
		id, err := strconv.Atoi(sid)
		if err == nil {
			var section models.Section
			database.DB.First(&section, uint(id))
			database.DB.Model(&post).Association("Sections").Append(&section)
		}
	}

	// Update tags
	database.DB.Model(&post).Association("Tags").Clear()
	tagNames := c.PostForm("tags")
	if tagNames != "" {
		tags := strings.Split(tagNames, ",")
		for _, tagName := range tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagSlug := utils.Slugify(tagName)
			var tag models.Tag
			result := database.DB.Where("slug = ?", tagSlug).First(&tag)
			if result.Error != nil {
				tag = models.Tag{Name: tagName, Slug: tagSlug}
				database.DB.Create(&tag)
			}
			database.DB.Model(&post).Association("Tags").Append(&tag)
		}
	}

	c.Redirect(http.StatusFound, AdminPath("/posts"))
}

func AdminPostDelete(c *gin.Context) {
	id := c.Param("id")
	database.DB.Delete(&models.Post{}, id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Category management
func AdminCategoryList(c *gin.Context) {
	var categories []models.Category
	database.DB.Find(&categories)
	c.HTML(http.StatusOK, "category_list.html", gin.H{
		"SiteName":   config.Load().SiteName,
		"Categories": categories,
	})
}

func AdminCategoryStore(c *gin.Context) {
	name := c.PostForm("name")
	slug := c.PostForm("slug")
	if slug == "" {
		slug = utils.Slugify(name)
	}
	if name != "" {
		database.DB.Create(&models.Category{Name: name, Slug: slug})
	}
	c.Redirect(http.StatusFound, AdminPath("/categories"))
}

func AdminCategoryDelete(c *gin.Context) {
	id := c.Param("id")
	database.DB.Delete(&models.Category{}, id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Comment management
func AdminCommentList(c *gin.Context) {
	var comments []models.Comment
	database.DB.Preload("Post").Preload("User").Order("created_at DESC").Find(&comments)
	c.HTML(http.StatusOK, "comment_list.html", gin.H{
		"SiteName": config.Load().SiteName,
		"Comments": comments,
	})
}

func AdminCommentApprove(c *gin.Context) {
	id := c.Param("id")
	database.DB.Model(&models.Comment{}).Where("id = ?", id).Update("status", "approved")
	c.Redirect(http.StatusFound, AdminPath("/comments"))
}

func AdminCommentReject(c *gin.Context) {
	id := c.Param("id")
	database.DB.Model(&models.Comment{}).Where("id = ?", id).Update("status", "rejected")
	c.Redirect(http.StatusFound, AdminPath("/comments"))
}

func AdminCommentDelete(c *gin.Context) {
	id := c.Param("id")
	database.DB.Delete(&models.Comment{}, id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
