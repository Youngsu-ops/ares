package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"youngsu-blog-plus/config"
)

// Allowed MIME types for upload validation
var allowedMimeTypes = map[string]map[string]bool{
	"image": {
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	},
	"video": {
		"video/mp4":  true,
		"video/webm": true,
	},
	"audio": {
		"audio/mpeg": true,
		"audio/wav":  true,
		"audio/ogg":  true,
		"audio/mp4":  true, // m4a
	},
}

// Allowed file extensions (SVG removed for security - SVG can contain scripts)
var validExts = map[string]map[string]bool{
	"image": {".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true},
	"video": {".mp4": true, ".webm": true, ".mov": true},
	"audio": {".mp3": true, ".wav": true, ".ogg": true, ".m4a": true},
}

func UploadFile(c *gin.Context) {
	cfg := config.Load()

	// Require type parameter - reject if missing or invalid
	mediaType := strings.ToLower(c.Query("type"))
	if mediaType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件类型参数 (type)"})
		return
	}
	if _, ok := validExts[mediaType]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型: " + mediaType})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "上传失败"})
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !validExts[mediaType][ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件格式: " + ext})
		return
	}

	// Validate MIME type (first 512 bytes) to prevent extension spoofing
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if n > 0 {
		detectedType := http.DetectContentType(buf[:n])
		// Extract base MIME type (e.g., "image/png" from "image/png; charset=binary")
		detectedType = strings.Split(detectedType, ";")[0]

		if allowed, ok := allowedMimeTypes[mediaType]; ok {
			if !allowed[detectedType] {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("文件内容类型不匹配: 检测到 %s", detectedType),
				})
				return
			}
		}

		// Reset file pointer for subsequent copy
		file.Seek(0, io.SeekStart)
	}

	// Security: limit file size
	maxSize := cfg.MaxUploadMB * 1024 * 1024
	if header.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("文件大小超过限制 (%dMB)", cfg.MaxUploadMB),
		})
		return
	}

	// Generate safe filename (no original name preserved)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	uploadDir := filepath.Join(cfg.UploadDir, mediaType+"s")

	// Ensure directory exists with restricted permissions
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	dst, err := os.OpenFile(filepath.Join(uploadDir, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}

	// Return the URL path
	url := fmt.Sprintf("/static/uploads/%s/%s", mediaType+"s", filename)
	c.JSON(http.StatusOK, gin.H{
		"url":      url,
		"filename": filename,
		"size":     header.Size,
	})
}
