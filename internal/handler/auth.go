// Package handler 负责处理 HTTP 请求（MVC 中的 Controller 层）。
//
// 每个 handler 的职责：
//   1. 解析请求参数（路径参数、查询参数、请求体 JSON）
//   2. 参数校验（Gin 的 binding tag 自动校验）
//   3. 调用 service 层的业务逻辑
//   4. 返回 HTTP 响应（状态码 + JSON）
//
// handler 不应该包含业务逻辑——这是分层架构的核心原则。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

// AuthHandler 认证相关的 HTTP 处理器。
// 持有 AuthService 实例，所有业务逻辑委托给它。
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 创建认证处理器实例。
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// registerRequest 注册请求的参数结构体。
//
// Gin 的 binding tag：
//
//	binding:"required"        → 字段必须存在且非零值
//	binding:"required,min=2"  → 必须存在且长度至少为 2
//	binding:"required,email"  → 必须存在且格式是合法邮箱
//
// 如果校验失败，c.ShouldBindJSON 返回错误，handler 直接返回 400。
type registerRequest struct {
	Username string `json:"username" binding:"required,min=2,max=64"` // 用户名，2-64 字符
	Email    string `json:"email" binding:"required,email"`            // 邮箱，必须符合邮箱格式
	Password string `json:"password" binding:"required,min=6"`         // 密码，至少 6 个字符
}

// loginRequest 登录请求的参数结构体。
type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`    // 登录邮箱
	Password string `json:"password" binding:"required"`        // 登录密码
}

// authResponse 认证成功后的响应结构体。
// 包含 JWT 令牌和用户基本信息。
type authResponse struct {
	Token string `json:"token"` // JWT 令牌，前端存储后在后续请求中携带
	User  struct {
		ID       uint64 `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

// Register 用户注册接口。
//
//	POST /api/auth/register
//	请求体：{"username": "张三", "email": "zhangsan@example.com", "password": "123456"}
//
// 处理流程：
//  1. 解析并校验请求参数
//  2. 检查邮箱是否已注册
//  3. 用 bcrypt 加密密码
//  4. 创建用户记录
//  5. 生成 JWT 令牌
//  6. 返回令牌和用户信息
func (h *AuthHandler) Register(c *gin.Context) {
	// Step 1: 解析 JSON 请求体，同时做参数校验
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 校验失败：可能是字段缺失、格式不对等
		// HTTP 400 Bad Request：客户端发送的数据有问题
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 2: 检查邮箱是否已被注册
	// First 查询第一条匹配记录，如果找到了说明邮箱已存在
	var existing model.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		// 找到了 → 邮箱已注册
		// HTTP 409 Conflict：资源冲突（重复注册）
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// Step 3: 加密密码
	hash, err := h.svc.HashPassword(req.Password)
	if err != nil {
		// HTTP 500 Internal Server Error：服务器内部错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Step 4: 创建用户记录
	user := model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash, // 存储的是加密后的哈希值，不是明文！
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Step 5: 生成 JWT 令牌
	token, err := h.svc.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Step 6: 组装并返回成功响应
	var resp authResponse
	resp.Token = token
	resp.User.ID = user.ID
	resp.User.Username = user.Username
	resp.User.Email = user.Email
	// HTTP 201 Created：资源创建成功
	c.JSON(http.StatusCreated, resp)
}

// Login 用户登录接口。
//
//	POST /api/auth/login
//	请求体：{"email": "zhangsan@example.com", "password": "123456"}
//
// 处理流程：
//  1. 解析并校验请求参数
//  2. 按邮箱查找用户
//  3. 用 bcrypt 验证密码
//  4. 生成 JWT 令牌
//  5. 返回令牌和用户信息
//
// 安全提示：登录失败时只返回 "invalid email or password"，
// 不具体说明是邮箱不存在还是密码错误。这样可以防止攻击者
// 通过错误信息判断某个邮箱是否已注册（用户枚举攻击）。
func (h *AuthHandler) Login(c *gin.Context) {
	// Step 1: 解析 JSON 请求体
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 2: 按邮箱查找用户
	var user model.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// 用户不存在（或数据库查询出错）
		// HTTP 401 Unauthorized：认证失败
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Step 3: 验证密码
	// bcrypt.CompareHashAndPassword 会用同样的算法计算哈希并比较
	if !h.svc.CheckPassword(req.Password, user.PasswordHash) {
		// 密码错误（同样返回模糊的错误信息）
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Step 4: 生成 JWT 令牌
	token, err := h.svc.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Step 5: 组装并返回成功响应
	var resp authResponse
	resp.Token = token
	resp.User.ID = user.ID
	resp.User.Username = user.Username
	resp.User.Email = user.Email
	c.JSON(http.StatusOK, resp)
}
