// Package config 负责管理应用的所有配置项。
//
// 配置来源：环境变量（推荐用于生产环境，避免把密码写死在代码里）
// 每个配置项都有默认值，方便本地开发时直接启动。
//
// 使用方式：
//
//	cfg := config.Load()
//	fmt.Println(cfg.ServerPort) // "8080"
//	fmt.Println(cfg.DSN())      // "root:@tcp(localhost:3306)/choex_manager?..."
package config

import (
	"os"
)

// Config 包含应用运行所需的所有配置项。
// 每个字段都对应一个环境变量，可以在 .env 文件或系统环境变量中设置。
type Config struct {
	// ---- 服务器配置 ----
	// ServerPort HTTP 服务监听的端口号，默认 8080
	ServerPort string

	// ---- MySQL 数据库配置 ----
	DBHost     string // 数据库主机地址，默认 localhost
	DBPort     string // 数据库端口，默认 3306
	DBUser     string // 数据库用户名，默认 root
	DBPassword string // 数据库密码，默认空
	DBName     string // 数据库名称，默认 choex_manager

	// ---- Redis 配置 ----
	RedisHost string // Redis 主机地址，默认 localhost
	RedisPort string // Redis 端口，默认 6379

	// ---- 安全相关 ----
	// JWTSecret 是签发和验证 JWT 令牌的密钥。
	// 生产环境务必修改为随机长字符串，否则令牌可被伪造。
	JWTSecret string

	// DeepSeekKey 是调用 DeepSeek 大模型 API 的密钥。
	// 从 https://platform.deepseek.com 获取。
	// 注意：这个字段没有默认值，必须通过环境变量设置。
	DeepSeekKey string

	// EncryptionKey 是 AES 加密密码时使用的密钥。
	// 必须是 32 字节（256 位），不足会自动填充。
	// 生产环境务必修改默认值！
	EncryptionKey string
}

// Load 从环境变量读取所有配置，返回一个 *Config 实例。
//
// 环境变量名和默认值的对应关系：
//
//	SERVER_PORT  → "8080"
//	DB_HOST      → "localhost"
//	DB_PORT      → "3306"
//	DB_USER      → "root"
//	DB_PASSWORD  → ""
//	DB_NAME      → "choex_manager"
//	REDIS_HOST   → "localhost"
//	REDIS_PORT   → "6379"
//	JWT_SECRET   → "dev-secret"（仅开发环境，生产务必修改！）
//	DEEPSEEK_API_KEY → 无默认值，必须设置
//	ENCRYPTION_KEY   → "choex2025-32byte-secret-key!!!"（仅开发环境）
func Load() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "3306"),
		DBUser:        getEnv("DB_USER", "root"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "choex_manager"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret"),
		DeepSeekKey:   os.Getenv("DEEPSEEK_API_KEY"), // 没有默认值，未设置时为空字符串
		EncryptionKey: getEnv("ENCRYPTION_KEY", "choex2025-32byte-secret-key!!!"),
	}
}

// DSN 拼装 MySQL 连接字符串（Data Source Name）。
//
// 格式：用户名:密码@tcp(主机:端口)/数据库名?charset=utf8mb4&parseTime=True&loc=Local
//
// 参数说明：
//   - charset=utf8mb4  使用 UTF-8 完整编码（支持 emoji 等 4 字节字符）
//   - parseTime=True   自动把数据库的 DATETIME 转为 Go 的 time.Time 类型
//   - loc=Local        时间使用服务器本地时区
func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

// RedisAddr 拼装 Redis 连接地址。
//
// 格式：主机:端口，例如 "localhost:6379"
func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// getEnv 是一个辅助函数，用于读取环境变量。
//
// 参数：
//   - key       环境变量名
//   - defaultVal 如果环境变量不存在或为空时使用的默认值
//
// 返回：环境变量的值（存在且非空）或默认值
//
// 示例：
//
//	getEnv("SERVER_PORT", "8080")
//	// 如果设置了 export SERVER_PORT=3000 → 返回 "3000"
//	// 如果没有设置                → 返回 "8080"
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
