package models

type Comment struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	PostID    uint   `gorm:"index;not null" json:"post_id"`
	Post      Post   `gorm:"foreignKey:PostID" json:"post,omitempty"`
	UserID    *uint  `gorm:"index" json:"user_id"`
	User      *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Author    string `json:"author"` // guest name if not registered
	Email     string `json:"email"`
	Content   string `gorm:"type:text;not null" json:"content"`
	Status    string `gorm:"default:pending" json:"status"` // pending, approved, rejected
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
