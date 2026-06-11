package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

// MemoryHandler 智能体记忆相关的 HTTP 处理器。
//
// 记忆管理功能允许用户：
//   - 创建多个"角色记忆"（不同场景用不同的 AI 人设）
//   - 切换激活的记忆（同一时间只有一个生效）
//   - 保存和管理记忆的多个版本（current/backup/custom）
//   - 恢复到之前的版本
type MemoryHandler struct {
	svc *service.MemoryService
}

// NewMemoryHandler 创建记忆处理器实例。
func NewMemoryHandler(svc *service.MemoryService) *MemoryHandler {
	return &MemoryHandler{svc: svc}
}

// List 获取当前用户的所有记忆。
//
//	GET /api/memories
func (h *MemoryHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	memories, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 确保返回空数组而非 null
	if memories == nil {
		memories = []model.AgentMemory{}
	}
	c.JSON(http.StatusOK, memories)
}

// Create 创建一条新记忆。
//
//	POST /api/memories
//	请求体：{"name": "理财助手", "icon": "💰"}
//
// 创建时会自动生成一个空内容的 "current" 版本。
func (h *MemoryHandler) Create(c *gin.Context) {
	var m model.AgentMemory
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m.UserID = c.GetUint64("user_id")
	if err := h.svc.Create(&m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

// Activate 激活指定记忆。
//
//	PUT /api/memories/:id/activate
//
// 激活后，该用户的其他所有记忆自动变为未激活状态。
// 同一时间只有一个记忆处于激活状态。
func (h *MemoryHandler) Activate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Activate(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "activated"})
}

// Delete 删除指定记忆及其所有版本。
//
//	DELETE /api/memories/:id
func (h *MemoryHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetVersions 获取指定记忆的所有版本。
//
//	GET /api/memories/:id/versions
//
// 返回格式：
//
//	{
//	  "current": "你是理财顾问...",
//	  "backup": "你是专业的理财助手...",
//	  "custom": "你擅长分析财务报表..."
//	}
func (h *MemoryHandler) GetVersions(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	versions, err := h.svc.GetVersions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

// SaveVersion 保存指定记忆的某个版本。
//
//	PUT /api/memories/:id/versions/:type
//	请求体：{"content": "你是专业的理财顾问，擅长分析股票和基金..."}
//
// URL 中的 :type 可以是 "current"、"backup"、"custom"。
//
// 特殊行为：保存 "current" 版本时，旧的 current 内容会被自动备份为 "backup"。
func (h *MemoryHandler) SaveVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	// c.Param("type") 获取路径中的 :type 占位符
	versionType := c.Param("type") // "current"、"backup" 或 "custom"
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SaveVersion(id, versionType, body.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// Restore 从 backup 版本恢复 current 版本。
//
//	PUT /api/memories/:id/restore
//
// 这个操作会把 backup 的内容复制到 current，
// 相当于"撤销"最近一次对 current 的修改。
func (h *MemoryHandler) Restore(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Restore(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "restored"})
}
