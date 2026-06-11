package model

import "time"

type Event struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64    `gorm:"index;not null" json:"user_id"`
	Title       string    `gorm:"size:256;not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Location    string    `gorm:"size:256" json:"location"`
	StartTime   time.Time `gorm:"not null" json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	AllDay      bool      `gorm:"default:false" json:"all_day"`
	Color       string    `gorm:"size:16" json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Event) TableName() string { return "events" }
