// Package main 是 Choex 个人生活管家服务端的程序入口。
//
// Choex 是一个 Web 后端服务，提供以下功能模块：
//   - 用户注册/登录（JWT 令牌认证）
//   - 日程管理（增删改查）
//   - 记账管理（收入/支出记录 + 按分类统计）
//   - 密码管理（AES-GCM 加密存储）
//   - AI 对话助手（接入 DeepSeek 大模型，支持流式回复）
//   - 智能体记忆（带版本管理，支持 current/backup/custom 三种版本）
//   - HTTP 代理（用于 iframe 嵌入外部页面）
//
// 技术栈：Go + Gin 框架 + GORM（MySQL ORM）+ Redis + JWT
//
// 启动流程：
//  1. 从环境变量加载配置
//  2. 连接 MySQL 和 Redis
//  3. 自动创建数据库表结构（AutoMigrate）
//  4. 注册所有路由和中间件
//  5. 启动 HTTP 服务器
package main

import (
	"log"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/router"
)

func main() {
	// ============================================================
	// 第一步：加载配置
	// 从环境变量中读取数据库地址、Redis 地址、JWT 密钥等配置项
	// 如果环境变量不存在，使用代码中预设的默认值
	// ============================================================
	cfg := config.Load()

	// ============================================================
	// 第二步：初始化数据库连接
	// 分别连接 MySQL（用 GORM）和 Redis（用 go-redis）
	// 连接失败会直接终止程序（log.Fatalf）
	// ============================================================
	database.Init(cfg)

	// ============================================================
	// 第三步：自动创建/更新数据库表结构
	//
	// AutoMigrate 的作用：
	//   - 如果表不存在 → 自动创建表
	//   - 如果表已存在但字段不同 → 自动添加新字段（不会删除已有字段）
	//   - 相当于"数据库表结构和 Go 结构体保持同步"
	//
	// 这里注册了 6 个数据模型对应的表：
	//   User           → users 表           用户信息
	//   Event          → events 表          日程事件
	//   Bill           → bills 表           账单记录
	//   Password       → passwords 表        加密密码
	//   AgentMemory    → agent_memories 表   智能体记忆
	//   MemoryVersion  → memory_versions 表  记忆版本
	// ============================================================
	if err := database.DB.AutoMigrate(
		&model.User{},
		&model.Event{},
		&model.Bill{},
		&model.Password{},
		&model.AgentMemory{},
		&model.MemoryVersion{},
	); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// ============================================================
	// 第四步：设置路由
	// 注册所有 API 路由、中间件（CORS、JWT 认证）
	// 返回配置好的 Gin 引擎实例
	// ============================================================
	r := router.Setup(cfg)

	// ============================================================
	// 第五步：启动 HTTP 服务器
	// 监听配置中指定的端口（默认 8080）
	// r.Run 会阻塞主线程，直到服务被手动停止
	// ============================================================
	log.Printf("Server running on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
