package models

type Tag struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;not null" json:"name"`
	Slug string `gorm:"uniqueIndex;not null" json:"slug"`
	Posts []Post `gorm:"many2many:post_tags;" json:"posts,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
