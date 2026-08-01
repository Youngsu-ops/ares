package models

type Post struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Title      string `gorm:"not null" json:"title"`
	Slug       string `gorm:"uniqueIndex;not null" json:"slug"`
	Content    string `gorm:"type:text" json:"content"`
	Excerpt    string `gorm:"type:text" json:"excerpt"`
	CoverImage string `json:"cover_image"`
	AudioURL   string `json:"audio_url"`
	VideoURL   string `json:"video_url"`
	CategoryID uint     `json:"category_id"`
	Category   Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Tags       []Tag    `gorm:"many2many:post_tags;" json:"tags,omitempty"`
	Sections   []Section `gorm:"many2many:section_posts;" json:"sections,omitempty"`
	Status     string `gorm:"default:draft" json:"status"` // draft, published
	Pinned     bool   `gorm:"default:false" json:"pinned"`
	Views      int    `gorm:"default:0" json:"views"`
	TimelineAt string `json:"timeline_at"` // custom position on timeline
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}
