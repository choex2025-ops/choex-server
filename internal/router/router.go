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
			protected.POST("/agent/chat", agentHandler.Chat)
		}
	}

	return r
}
