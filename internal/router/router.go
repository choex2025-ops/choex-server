// Package router 负责注册所有的 HTTP 路由和中间件。
//
// 路由是什么？
//
//	路由决定了"哪个 URL 由哪个函数处理"。
//	例如：POST /api/auth/login → authHandler.Login
//	      GET  /api/events     → calendarHandler.List
//
// 本项目使用 Gin 框架的路由系统，支持：
//   - 路由分组（Group）：把相关接口放在一起，统一应用中间件
//   - 路径参数（:id）：动态匹配 URL 中的 ID
//   - 中间件链：分组级别/单个路由级别的中间件
//
// 路由结构概览：
//
//	/api
//	├── /auth                        (公开接口，不需要登录)
//	│   ├── POST /register           注册
//	│   └── POST /login              登录
//	│
//	├── (protected)                  (以下接口需要 JWT 认证)
//	│   ├── POST /agent/chat         AI 对话（SSE 流式响应）
//	│   ├── /events                  日程管理 CRUD
//	│   ├── /bills                   记账管理 + 统计
//	│   ├── /passwords               密码管理 CRUD
//	│   └── /memories                记忆管理 + 版本控制
//	│
//	└── GET /proxy                   浏览器代理（不需认证）
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/agent"
	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/handler"
	"github.com/choex2025-ops/choex-server/internal/middleware"
	"github.com/choex2025-ops/choex-server/internal/service"
)

// Setup 创建并配置 Gin 路由器。
//
// 这个函数在程序启动时调用一次（见 main.go），返回配置好的路由器实例。
// 所有依赖（service、handler）在这里统一创建和注入——这就是"依赖注入"的雏形。
//
// 参数：
//   - cfg：应用配置
//
// 返回：配置好的 *gin.Engine 实例
func Setup(cfg *config.Config) *gin.Engine {
	// gin.Default() 创建默认的 Gin 引擎，自动带 Logger 和 Recovery 中间件
	//   - Logger：记录每个请求的耗时、状态码等
	//   - Recovery：捕获 handler 中的 panic，返回 500 而不是让进程崩溃
	r := gin.Default()

	// ---- 全局中间件 ----
	// CORS 中间件应用于所有请求（因为跨域检测需要在所有响应头中添加）
	r.Use(middleware.CORS())

	// ---- 创建依赖 ----
	// 按 配置 → Service → Handler 的顺序创建
	// 每个 Handler 持有对应的 Service 实例，Service 持有配置或数据库连接

	// 认证模块
	authSvc := service.NewAuthService(cfg)
	authHandler := handler.NewAuthHandler(authSvc)

	// AI 对话模块（agent 比较特殊，它自己就是 handler + 业务逻辑的结合）
	agentHandler := agent.NewAgentHandler(cfg)

	// 日程模块
	calendarSvc := service.NewCalendarService()
	calendarHandler := handler.NewCalendarHandler(calendarSvc)

	// 记账模块
	billSvc := service.NewBillService()
	billHandler := handler.NewBillHandler(billSvc)

	// 密码管理模块（需要传入加密密钥）
	passwordSvc := service.NewPasswordService(cfg.EncryptionKey)
	passwordHandler := handler.NewPasswordHandler(passwordSvc)

	// 记忆管理模块
	memorySvc := service.NewMemoryService()
	memoryHandler := handler.NewMemoryHandler(memorySvc)

	// ---- 注册路由 ----
	// r.Group("/api") 创建 /api 路径前缀的路由组
	// 组内所有路由都会自动加上 /api 前缀
	api := r.Group("/api")
	{
		// --- 公开接口（不需要认证） ---
		auth := api.Group("/auth")
		{
			// POST /api/auth/register → authHandler.Register
			auth.POST("/register", authHandler.Register)
			// POST /api/auth/login → authHandler.Login
			auth.POST("/login", authHandler.Login)
		}

		// --- 受保护接口（需要 JWT 认证） ---
		// api.Group("") 创建一个空路径的子组（路径前缀为空）
		// .Use(middleware.AuthRequired(authSvc)) 给这个组的所有路由加认证中间件
		protected := api.Group("")
		protected.Use(middleware.AuthRequired(authSvc))
		{
			// === AI 对话 ===
			// POST /api/agent/chat → agentHandler.Chat（SSE 流式响应）
			protected.POST("/agent/chat", agentHandler.Chat)

			// === 日程管理 ===
			// 标准 RESTful CRUD 路由
			protected.GET("/events", calendarHandler.List)          // 列表
			protected.POST("/events", calendarHandler.Create)        // 创建
			protected.PUT("/events/:id", calendarHandler.Update)     // 更新（:id 是路径参数）
			protected.DELETE("/events/:id", calendarHandler.Delete)  // 删除

			// === 记账管理 ===
			protected.GET("/bills", billHandler.List)                // 列表（支持 ?date= 筛选）
			protected.POST("/bills", billHandler.Create)             // 创建
			protected.DELETE("/bills/:id", billHandler.Delete)       // 删除
			protected.GET("/bills/stats", billHandler.Stats)         // 统计（需要 ?month=YYYY-MM）

			// === 密码管理 ===
			protected.GET("/passwords", passwordHandler.List)        // 列表（不含明文密码）
			protected.POST("/passwords", passwordHandler.Create)     // 创建
			protected.GET("/passwords/:id", passwordHandler.Get)     // 查看单条（含明文密码）
			protected.PUT("/passwords/:id", passwordHandler.Update)  // 更新
			protected.DELETE("/passwords/:id", passwordHandler.Delete) // 删除

			// === 记忆管理 ===
			protected.GET("/memories", memoryHandler.List)                    // 列表
			protected.POST("/memories", memoryHandler.Create)                 // 创建
			protected.PUT("/memories/:id/activate", memoryHandler.Activate)   // 激活
			protected.DELETE("/memories/:id", memoryHandler.Delete)           // 删除
			protected.GET("/memories/:id/versions", memoryHandler.GetVersions) // 获取所有版本
			// :type 可以是 "current"、"backup"、"custom"
			protected.PUT("/memories/:id/versions/:type", memoryHandler.SaveVersion) // 保存版本
			protected.PUT("/memories/:id/restore", memoryHandler.Restore)            // 恢复备份
		}

		// --- 代理接口（不需要认证，用于 iframe 嵌入） ---
		// GET /api/proxy?url=https://example.com
		api.GET("/proxy", handler.ProxyHandler)
	}

	return r
}
