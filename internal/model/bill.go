package model

import "time"

// Bill 代表一条记账记录，对应数据库的 bills 表。
//
// 支持两种类型：
//   - income（收入）：如工资、奖金、退款等
//   - expense（支出）：如餐饮、交通、购物等
//
// 每条账单可以标记分类和备注，方便后续按分类统计。
// 账单日期用字符串存储（格式：YYYY-MM-DD），而非 time.Time，
// 这样前端可以直接用日期选择器的值，不用处理时区转换。
type Bill struct {
	// ID 账单唯一标识，自增主键
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// UserID 所属用户的 ID
	UserID uint64 `gorm:"index;not null" json:"user_id"`

	// Amount 金额（元）
	// 使用 DECIMAL(10,2) 而非 FLOAT，避免浮点数精度问题
	// 例如：0.1 + 0.2 在浮点数中可能等于 0.30000000000000004
	// DECIMAL 存储的是精确值，适合金额计算
	Amount float64 `gorm:"type:decimal(10,2);not null" json:"amount"`

	// Type 账单类型："income"（收入）或 "expense"（支出）
	Type string `gorm:"size:16;not null" json:"type"`

	// Category 分类标签，如"餐饮"、"交通"、"购物"、"工资"
	Category string `gorm:"size:64" json:"category"`

	// Note 备注说明，如"和同事 AA 午餐"、"地铁通勤"
	Note string `gorm:"size:256" json:"note"`

	// BillDate 账单日期，格式：YYYY-MM-DD（如 "2025-06-11"）
	// 使用字符串而非 time.Time 可以避免时区问题
	BillDate string `gorm:"size:10;not null" json:"bill_date"`

	// CreatedAt 创建时间，GORM 自动填充
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间，GORM 自动维护
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名为 "bills"。
func (Bill) TableName() string { return "bills" }
