package models

import "golang.org/x/crypto/bcrypt"

type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"uniqueIndex;not null" json:"username"`
	Password  string `gorm:"not null" json:"-"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Bio       string `json:"bio"`
	Role      string `gorm:"default:reader" json:"role"` // admin, reader, vip
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func HashPassword(pwd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(hash, pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)) == nil
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) CanAccessSection(sectionIDs []uint, purchasedSectionIDs map[uint]bool) bool {
	if u.IsAdmin() {
		return true
	}
	for _, sid := range sectionIDs {
		if purchasedSectionIDs[sid] {
			return true
		}
	}
	return false
}
