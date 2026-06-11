package router

import (
	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/agent"
	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/handler"
	"github.com/choex2025-ops/choex-server/internal/middleware"
	"github.com/choex2025-ops/choex-server/internal/service"
)

func Setup(cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	authSvc := service.NewAuthService(cfg)
	authHandler := handler.NewAuthHandler(authSvc)
	agentHandler := agent.NewAgentHandler(cfg)

	calendarSvc := service.NewCalendarService()
	calendarHandler := handler.NewCalendarHandler(calendarSvc)

	billSvc := service.NewBillService()
	billHandler := handler.NewBillHandler(billSvc)

	passwordSvc := service.NewPasswordService(cfg.EncryptionKey)
	passwordHandler := handler.NewPasswordHandler(passwordSvc)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(authSvc))
		{
			// Agent
			protected.POST("/agent/chat", agentHandler.Chat)

			// Calendar
			protected.GET("/events", calendarHandler.List)
			protected.POST("/events", calendarHandler.Create)
			protected.PUT("/events/:id", calendarHandler.Update)
			protected.DELETE("/events/:id", calendarHandler.Delete)

			// Bills
			protected.GET("/bills", billHandler.List)
			protected.POST("/bills", billHandler.Create)
			protected.DELETE("/bills/:id", billHandler.Delete)
			protected.GET("/bills/stats", billHandler.Stats)

			// Passwords
			protected.GET("/passwords", passwordHandler.List)
			protected.POST("/passwords", passwordHandler.Create)
			protected.GET("/passwords/:id", passwordHandler.Get)
			protected.PUT("/passwords/:id", passwordHandler.Update)
			protected.DELETE("/passwords/:id", passwordHandler.Delete)

			// Browser proxy
			protected.GET("/proxy", handler.ProxyHandler)
		}
	}

	return r
}
