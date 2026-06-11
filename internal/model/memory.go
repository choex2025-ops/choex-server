package model

import "time"

// AgentMemory 代表一个智能体记忆，对应数据库的 agent_memories 表。
//
// 智能体记忆是什么？
//   在和 AI 对话时，可以给 AI 设定不同的"角色记忆"（也叫 system prompt / 人设）。
//   例如：
//     - "你是一个擅长理财的个人助手" → 记忆内容里写理财相关的指令
//     - "你是一个日程规划师"         → 记忆内容里写规划相关的指令
//
// 用户可以创建多个记忆，但同一时间只有一个处于激活状态（IsActive = true）。
// 切换激活记忆时，之前激活的会自动取消。
//
// 每个记忆可以保存多个版本（见 MemoryVersion），支持备份和恢复。
type AgentMemory struct {
	// ID 记忆唯一标识，自增主键
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// UserID 所属用户的 ID
	UserID uint64 `gorm:"index;not null" json:"user_id"`

	// Name 记忆名称，如"理财助手"、"日程管家"
	Name string `gorm:"size:64;not null" json:"name"`

	// Icon 记忆图标，存储 emoji 字符，如"💰"、"📅"
	Icon string `gorm:"size:16" json:"icon"`

	// IsActive 是否为当前激活的记忆
	// 同一用户同一时间只有一个记忆是激活状态
	IsActive bool `gorm:"default:false" json:"is_active"`

	// CreatedAt 创建时间，GORM 自动填充
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间，GORM 自动维护
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名为 "agent_memories"。
func (AgentMemory) TableName() string { return "agent_memories" }

// MemoryVersion 代表记忆的一个版本快照，对应数据库的 memory_versions 表。
//
// 版本类型（VersionType）：
//   - "current"：当前正在使用的版本（每个记忆只有 1 个 current）
//   - "backup" ：自动备份的上一版本（保存 current 时自动备份）
//   - "custom" ：用户手动保存的自定义版本
//
// 使用场景：
//   1. 用户修改了 AI 的 system prompt，系统自动把旧版本备份为 "backup"
//   2. 用户可以手动保存某个版本为 "custom"，方便在多个版本间切换
//   3. 如果新版本效果不好，可以用 Restore 恢复到 backup 版本
type MemoryVersion struct {
	// ID 版本唯一标识，自增主键
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// MemoryID 关联的记忆 ID（外键关系）
	MemoryID uint64 `gorm:"index;not null" json:"memory_id"`

	// VersionType 版本类型："current"、"backup"、"custom"
	VersionType string `gorm:"size:16;not null" json:"version_type"`

	// Content 记忆的具体内容（即 system prompt 文本）
	// TEXT 类型，可以存储很长的提示词
	Content string `gorm:"type:text" json:"content"`

	// CreatedAt 创建时间，GORM 自动填充
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名为 "memory_versions"。
func (MemoryVersion) TableName() string { return "memory_versions" }
