package model

import "time"

type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"size:64;not null" json:"username"`
	Email        string    `gorm:"size:128;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:256;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
