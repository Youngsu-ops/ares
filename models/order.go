package models

type Order struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	UserID    uint    `gorm:"index;not null" json:"user_id"`
	SectionID uint    `gorm:"index;not null" json:"section_id"`
	Amount    float64 `json:"amount"`
	Status    string  `gorm:"default:pending" json:"status"` // pending, paid, cancelled
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}
