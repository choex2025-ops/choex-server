package model

import "time"

// Event 代表一个日程事件，对应数据库的 events 表。
//
// 每个事件属于某个用户（通过 UserID 关联），包含标题、时间、地点等信息。
// 支持全天事件标记和颜色标记，方便在前端日历组件中展示。
//
// 典型的 RESTful 操作：
//   GET    /api/events          → 获取当前用户的所有事件
//   POST   /api/events          → 创建新事件
//   PUT    /api/events/:id      → 更新指定事件
//   DELETE /api/events/:id      → 删除指定事件
type Event struct {
	// ID 事件唯一标识，自增主键
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// UserID 所属用户的 ID
	// 有普通索引（非唯一），因为一个用户可以有多个事件
	// 查询时通过这个字段过滤：WHERE user_id = ?
	UserID uint64 `gorm:"index;not null" json:"user_id"`

	// Title 事件标题，如"团队周会"、"去看牙医"，最长 256 字符
	Title string `gorm:"size:256;not null" json:"title"`

	// Description 事件详细描述，TEXT 类型，可存储长文本
	Description string `gorm:"type:text" json:"description"`

	// Location 事件地点，如"3 楼会议室 A"、"朝阳区中心医院"
	Location string `gorm:"size:256" json:"location"`

	// StartTime 事件开始时间，精确到秒
	StartTime time.Time `gorm:"not null" json:"start_time"`

	// EndTime 事件结束时间，精确到秒
	EndTime time.Time `json:"end_time"`

	// AllDay 是否为全天事件
	// 全天事件在前端日历中显示在日期顶部，不占具体时间段
	AllDay bool `gorm:"default:false" json:"all_day"`

	// Color 事件颜色标记，如 "#FF6B6B"（红色）、"#4ECDC4"（青色）
	// 前端用这个颜色来区分不同类型的事件
	Color string `gorm:"size:16" json:"color"`

	// CreatedAt 创建时间，GORM 自动填充
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间，GORM 自动维护
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名为 "events"。
func (Event) TableName() string { return "events" }
