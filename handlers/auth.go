package handlers

import (
	"net/http"
	"regexp"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
	"youngsu-blog-plus/database"
	"youngsu-blog-plus/models"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ShowLogin shows login page
func ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"SiteName": config.Load().SiteName,
	})
}

// ShowRegister shows registration page
func ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"SiteName": config.Load().SiteName,
	})
}

// Login handles user login
func Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "用户名或密码错误",
		})
		return
	}

	if !models.CheckPassword(user.Password, password) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "用户名或密码错误",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Save()

	if user.IsAdmin() {
		c.Redirect(http.StatusFound, AdminPath("/dashboard"))
		return
	}
	c.Redirect(http.StatusFound, "/")
}

// Register handles user registration
func Register(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	confirmPass := c.PostForm("confirm_password")
	email := c.PostForm("email")

	// Validation
	if len(username) < 3 || len(username) > 32 {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "用户名长度需在 3-32 个字符之间",
		})
		return
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(username) {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "用户名只能包含字母、数字和下划线",
		})
		return
	}

	if len(password) < 6 {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "密码长度至少为6位",
		})
		return
	}

	if password != confirmPass {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "两次密码输入不一致",
		})
		return
	}

	if email != "" && !emailRegex.MatchString(email) {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "邮箱格式不正确",
		})
		return
	}

	// Check duplicates
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "用户名已存在",
		})
		return
	}

	hashed, _ := models.HashPassword(password)
	user := models.User{
		Username: username,
		Password: hashed,
		Email:    email,
		Role:     "reader",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"SiteName": config.Load().SiteName,
			"Error":    "注册失败，请重试",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Save()

	c.Redirect(http.StatusFound, "/")
}

// Logout handles user logout
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/")
}
