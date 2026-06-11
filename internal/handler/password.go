package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

// PasswordHandler 密码管理相关的 HTTP 处理器。
type PasswordHandler struct {
	svc *service.PasswordService
}

// NewPasswordHandler 创建密码管理处理器实例。
func NewPasswordHandler(svc *service.PasswordService) *PasswordHandler {
	return &PasswordHandler{svc: svc}
}

// passwordBody 密码记录创建/更新的请求体。
// 注意：这里包含明文密码字段，因为用户输入的是明文，
// 加密操作在 handler 中完成后再传给 service。
type passwordBody struct {
	Title    string `json:"title"`    // 标题，如"GitHub"
	URL      string `json:"url"`      // 网站地址
	Username string `json:"username"` // 登录用户名
	Password string `json:"password"` // 明文密码（会加密后存储）
	Note     string `json:"note"`     // 备注
	Category string `json:"category"` // 分类："工作"、"个人"、"金融"等
}

// List 获取当前用户的所有密码记录（不含明文密码）。
//
//	GET /api/passwords
//
// 安全设计：列表接口不返回密码字段，只有点击查看单条时才返回明文。
func (h *PasswordHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	passwords, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 确保返回空数组而非 null
	if passwords == nil {
		passwords = []model.Password{}
	}
	c.JSON(http.StatusOK, passwords)
}

// Create 创建一条密码记录。
//
//	POST /api/passwords
//	请求体：{"title": "GitHub", "url": "https://github.com", "username": "myuser", "password": "secret123"}
//
// 密码处理流程：
//
//	用户输入的明文 "secret123"
//	  → AES-256-GCM 加密
//	  → Base64 编码
//	  → 存入数据库的 encrypted_password 字段
//
// 数据库里存的不是 "secret123"，而是一串像乱码的字符串。
func (h *PasswordHandler) Create(c *gin.Context) {
	var body passwordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 加密明文密码
	encrypted, err := h.svc.Encrypt(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	// 组装数据（用 map 而非 model.Password 结构体，
	// 因为 model.Password.EncryptedPassword 有 json:"-" 标签）
	p := map[string]any{
		"user_id":            c.GetUint64("user_id"),
		"title":              body.Title,
		"url":                body.URL,
		"username":           body.Username,
		"encrypted_password": encrypted, // 存储加密后的密文
		"note":               body.Note,
		"category":           body.Category,
	}

	if err := h.svc.CreateRaw(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 创建成功后返回基本信息（不返回加密密码）
	c.JSON(http.StatusCreated, gin.H{"title": body.Title, "username": body.Username, "url": body.URL})
}

// Get 获取单条密码记录，包含解密后的明文密码。
//
//	GET /api/passwords/:id
//
// 只有记录的所有者才能查看（由 service 层的 user_id 校验保证）。
func (h *PasswordHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	userID := c.GetUint64("user_id")
	p, err := h.svc.Get(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// Update 更新指定密码记录。
//
//	PUT /api/passwords/:id
//	请求体：{"title": "新标题", "password": "new_secret"}  （只传要修改的字段）
//
// 注意：如果传了 password 字段（非空），会先加密再更新。
// 如果没传 password（空字符串），则只更新其他字段，密码保持不变。
func (h *PasswordHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")

	var body passwordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先组装不包含密码的更新数据
	updates := map[string]any{
		"title":    body.Title,
		"url":      body.URL,
		"username": body.Username,
		"note":     body.Note,
		"category": body.Category,
	}
	// 如果传了新密码，加密后加入更新数据
	if body.Password != "" {
		encrypted, err := h.svc.Encrypt(body.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
			return
		}
		updates["encrypted_password"] = encrypted
	}

	if err := h.svc.Update(id, userID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete 删除指定密码记录。
//
//	DELETE /api/passwords/:id
func (h *PasswordHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
