// Package model 定义所有的数据模型（数据库表结构）。
//
// 每个 .go 文件定义一个或多个结构体，每个结构体对应数据库中的一张表。
// 结构体的字段通过 tag（反引号里的内容）来描述数据库列的类型、约束等。
//
// GORM 的 tag 约定：
//   - gorm:"primaryKey"             → 主键
//   - gorm:"autoIncrement"          → 自增
//   - gorm:"size:128"               → 列最大长度 128
//   - gorm:"uniqueIndex"            → 唯一索引（不允许重复值）
//   - gorm:"not null"               → 不允许为空
//   - gorm:"default:false"          → 默认值为 false
//   - gorm:"type:text"              → 使用 TEXT 类型（长文本）
//   - gorm:"type:decimal(10,2)"     → 使用 DECIMAL 类型（精确小数）
//
// JSON tag 约定：
//   - json:"field_name"             → JSON 序列化时使用这个名字
//   - json:"-"                      → JSON 序列化时忽略这个字段（不输出到前端）
package model

import "time"

// User 代表一个注册用户，对应数据库的 users 表。
//
// 用户通过邮箱注册，密码用 bcrypt 算法加密存储（不可逆），
// 登录成功后服务端返回 JWT 令牌，后续请求需携带令牌。
type User struct {
	// ID 用户唯一标识，自增主键
	// 使用 uint64（无符号 64 位整数），最大值约 1.8 × 10^19，足够用
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// Username 用户昵称/显示名，最长 64 个字符
	Username string `gorm:"size:64;not null" json:"username"`

	// Email 登录邮箱，有唯一索引确保不重复
	// 唯一索引的好处：1) 防止重复注册 2) 加速按邮箱查询
	Email string `gorm:"size:128;uniqueIndex;not null" json:"email"`

	// PasswordHash bcrypt 加密后的密码哈希值
	// json:"-" 表示返回 JSON 给前端时永远不包含这个字段（安全考虑）
	// bcrypt 的特点：同样的密码每次加密结果不同，无法反推原密码
	PasswordHash string `gorm:"size:256;not null" json:"-"`

	// CreatedAt 账号创建时间，GORM 自动填充
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 最近更新时间，GORM 自动维护
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定这个模型对应的数据库表名。
// 如果不写这个方法，GORM 默认会把结构体名转为蛇形命名复数形式（User → users）。
// 显式写出来更清晰，也避免命名规则变化导致的表名不一致。
func (User) TableName() string {
	return "users"
}
