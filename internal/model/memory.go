package model

import "time"

type AgentMemory struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Icon      string    `gorm:"size:16" json:"icon"`
	IsActive  bool      `gorm:"default:false" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AgentMemory) TableName() string { return "agent_memories" }

type MemoryVersion struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	MemoryID    uint64    `gorm:"index;not null" json:"memory_id"`
	VersionType string    `gorm:"size:16;not null" json:"version_type"` // current, backup, custom
	Content     string    `gorm:"type:text" json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

func (MemoryVersion) TableName() string { return "memory_versions" }
