package model

import "time"

// Password 代表一条密码记录，对应数据库的 passwords 表。
//
// 这是一个简易密码管理器，用户可以存储网站/应用的登录凭据。
// 密码使用 AES-256-GCM 加密存储，只有服务端有密钥才能解密。
//
// 安全设计要点：
//   1. EncryptedPassword 字段用 json:"-" 标记，JSON 序列化时不会输出
//   2. 只有调用 Get 获取单条记录时，才解密并返回明文密码
//   3. 列表查询时不返回密码字段
type Password struct {
	// ID 密码记录唯一标识，自增主键
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// UserID 所属用户的 ID
	UserID uint64 `gorm:"index;not null" json:"user_id"`

	// Title 记录标题，如"公司邮箱"、"GitHub"、"银行登录"
	Title string `gorm:"size:128;not null" json:"title"`

	// URL 网站地址，如"https://github.com/login"
	URL string `gorm:"size:512" json:"url"`

	// Username 登录用户名/邮箱
	Username string `gorm:"size:128" json:"username"`

	// EncryptedPassword 加密后的密码密文（AES-256-GCM + Base64）
	// json:"-" 非常重要：即使不小心序列化了整个结构体，密码也不会泄露
	EncryptedPassword string `gorm:"type:text;not null" json:"-"`

	// Note 备注，如"两步验证已开启"、"支付密码，非登录密码"
	Note string `gorm:"size:256" json:"note"`

	// Category 分类标签，如"工作"、"个人"、"金融"
	Category string `gorm:"size:64" json:"category"`

	// CreatedAt 创建时间，GORM 自动填充
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间，GORM 自动维护
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名为 "passwords"。
func (Password) TableName() string { return "passwords" }
