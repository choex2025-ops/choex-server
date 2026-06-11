// Package database 负责数据库和缓存的连接管理。
//
// 本包初始化两个全局单例：
//   - DB：MySQL 数据库连接（基于 GORM，一个 Go 语言最流行的 ORM 库）
//   - Redis：Redis 缓存连接（基于 go-redis）
//
// ORM（Object-Relational Mapping）是什么？
//
//	简单说就是把"数据库表的一行"和"Go 结构体的一个实例"对应起来。
//	比如：DB.Create(&user) 就相当于 INSERT INTO users VALUES (...)
//	      DB.First(&user, 1) 就相当于 SELECT * FROM users WHERE id = 1
//	不用手写 SQL，也不用手动把查询结果转换成结构体。
//
// 使用方式（在别的包中）：
//
//	import "github.com/choex2025-ops/choex-server/internal/database"
//	database.DB.Where("user_id = ?", 1).Find(&events)
package database

import (
	"context"
	"log"

	"gorm.io/driver/mysql" // GORM 的 MySQL 驱动
	"gorm.io/gorm"
	"gorm.io/gorm/logger" // GORM 的日志级别控制

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/redis/go-redis/v9" // Redis Go 客户端
)

// 全局变量：在整个应用中共享同一个数据库连接和 Redis 连接。
// 用 var（而不是 const）是因为需要在 Init 函数中赋值。
var (
	DB    *gorm.DB       // MySQL 数据库连接实例，所有 SQL 操作都通过它
	Redis *redis.Client   // Redis 客户端实例，用于缓存操作
)

// Init 初始化数据库和 Redis 连接。
//
// 这个函数只在程序启动时调用一次（见 main.go）。
// 如果连接失败，直接终止程序——因为数据库不可用时服务无法正常工作。
//
// 参数：
//   - cfg：应用配置，包含数据库地址、Redis 地址等
func Init(cfg *config.Config) {
	var err error

	// ---- 连接 MySQL ----
	// gorm.Open 的两个参数：
	//   1. mysql.Open(dsn)：告诉 GORM 用 MySQL 驱动，dsn 是连接字符串
	//   2. &gorm.Config{Logger: ...}：设置日志级别为 Info
	//      - Silent：不打印任何 SQL
	//      - Error：只打印出错的 SQL
	//      - Warn：打印慢 SQL（默认）
	//      - Info：打印所有 SQL（开发时推荐，方便调试）
	DB, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 打印每条 SQL 语句到控制台
	})
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// ---- 连接 Redis ----
	// Redis 在这里用于缓存和会话管理（当前版本主要用于基础连接）
	// redis.NewClient 创建客户端，但不会立即连接（懒连接模式）
	// 所以需要 Ping 一下确认能连通
	Redis = redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr(), // Redis 地址，如 "localhost:6379"
	})
	// Ping 发送一个 PING 命令到 Redis，如果 Redis 正常会返回 PONG
	if err := Redis.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Database and Redis connected")
}
