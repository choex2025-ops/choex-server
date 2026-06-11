package model

import "time"

type Password struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint64    `gorm:"index;not null" json:"user_id"`
	Title             string    `gorm:"size:128;not null" json:"title"`
	URL               string    `gorm:"size:512" json:"url"`
	Username          string    `gorm:"size:128" json:"username"`
	EncryptedPassword string    `gorm:"type:text;not null" json:"-"`
	Note              string    `gorm:"size:256" json:"note"`
	Category          string    `gorm:"size:64" json:"category"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Password) TableName() string { return "passwords" }
