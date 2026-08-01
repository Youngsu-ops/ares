package models

type Section struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Name        string  `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string  `gorm:"uniqueIndex;not null" json:"slug"`
	Description string  `gorm:"type:text" json:"description"`
	CoverImage  string  `json:"cover_image"`
	Price       float64 `gorm:"default:0" json:"price"` // 价格，0表示免费
	Posts       []Post  `gorm:"many2many:section_posts;" json:"posts,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}
