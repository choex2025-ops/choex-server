// Package middleware 包含 Gin 框架的中间件。
//
// 中间件是什么？
//
//	中间件是 HTTP 请求处理链中的一个环节，在请求到达 handler 之前（或之后）执行。
//	可以理解为一道"关卡"，每个请求都要先通过中间件，才能到达 handler。
//
//	请求处理流程：
//	  客户端 → [CORS 中间件] → [Auth 中间件] → [Handler 处理函数] → 返回响应
//
//	Gin 中间件的典型用法：
//	  r.Use(middlewareFunc)            // 全局中间件（所有请求都经过）
//	  group.Use(middlewareFunc)        // 分组中间件（只有该组的请求经过）
//
//	中间件函数签名：func(c *gin.Context)
//	  调用 c.Next() → 继续执行下一个中间件/handler
//	  调用 c.Abort() → 中断处理链，直接返回响应
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/service"
)

// AuthRequired 返回一个 JWT 认证中间件。
//
// 这个中间件检查请求头中的 Authorization 字段，验证 JWT 令牌的有效性，
// 并将解析出的用户信息注入到请求上下文中。
//
// 认证流程：
//  1. 检查 Authorization 头部是否存在且以 "Bearer " 开头
//  2. 提取令牌（去掉 "Bearer " 前缀）
//  3. 解析并验证 JWT 令牌（签名是否正确、是否过期）
//  4. 将 user_id 和 email 注入 Gin Context
//  5. 调用 c.Next() 继续处理
//
// 如果任一步失败，返回 401 Unauthorized 并中断请求。
//
// 参数：
//   - svc：认证服务实例（用于解析 JWT 令牌）
//
// 返回：gin.HandlerFunc（中间件函数）
func AuthRequired(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: 获取 Authorization 请求头
		header := c.GetHeader("Authorization")

		// Step 2: 检查头部格式
		// JWT 标准格式：Authorization: Bearer <token>
		// 如果头部不存在，或者不是以 "Bearer " 开头，返回 401
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return // c.AbortWithStatusJSON 不会中断函数，需要手动 return
		}

		// Step 3: 提取令牌内容（去掉 "Bearer " 前缀）
		token := strings.TrimPrefix(header, "Bearer ")

		// Step 4: 解析并验证 JWT 令牌
		// ParseToken 会验证：
		//   - 签名是否正确（防止令牌被篡改）
		//   - 是否过期（exp 字段）
		//   - 格式是否合法
		claims, err := svc.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Step 5: 将用户信息注入到 Gin Context
		// 后续的 handler 可以通过 c.GetUint64("user_id") 和 c.GetString("email") 获取
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		// Step 6: 继续处理链
		// c.Next() 表示"我的工作做完了，交给下一个中间件或 handler"
		c.Next()
	}
}
