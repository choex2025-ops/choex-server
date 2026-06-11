package model

import "time"

type Bill struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index;not null" json:"user_id"`
	Amount    float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Type      string    `gorm:"size:16;not null" json:"type"`
	Category  string    `gorm:"size:64" json:"category"`
	Note      string    `gorm:"size:256" json:"note"`
	BillDate  string    `gorm:"size:10;not null" json:"bill_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Bill) TableName() string { return "bills" }
