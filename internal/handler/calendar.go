package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

// CalendarHandler 日程相关的 HTTP 处理器。
type CalendarHandler struct {
	svc *service.CalendarService
}

// NewCalendarHandler 创建日程处理器实例。
func NewCalendarHandler(svc *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{svc: svc}
}

// List 获取当前用户的所有日程。
//
//	GET /api/events
//
// 返回的 user_id 是从 JWT 令牌中提取的（由 Auth 中间件注入到上下文）。
// 这确保了用户只能看到自己的日程。
func (h *CalendarHandler) List(c *gin.Context) {
	// c.GetUint64("user_id") 从 Gin 上下文中取出中间件注入的 user_id
	// 这个值是在 AuthRequired 中间件中通过 c.Set("user_id", claims.UserID) 设置的
	userID := c.GetUint64("user_id")
	events, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 确保返回的是空数组 [] 而不是 null
	// 前端代码通常期望数组类型（方便直接调用 .forEach、.map 等方法）
	if events == nil {
		events = []model.Event{}
	}
	c.JSON(http.StatusOK, events)
}

// Create 创建一条新日程。
//
//	POST /api/events
//	请求体：{"title": "团队周会", "start_time": "2025-06-11T09:00:00Z", ...}
//
// 注意：UserID 不从客户端接收，而是从 JWT 令牌中获取。
// 这样做是为了安全——防止用户 A 冒充用户 B 创建日程。
func (h *CalendarHandler) Create(c *gin.Context) {
	var event model.Event
	// ShouldBindJSON 把 JSON 请求体映射到 event 结构体
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 重要：覆盖 UserID 为当前登录用户，忽略客户端传的值
	event.UserID = c.GetUint64("user_id")
	if err := h.svc.Create(&event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// HTTP 201 Created：资源创建成功，返回创建后的完整记录（包含数据库生成的 ID）
	c.JSON(http.StatusCreated, event)
}

// Update 更新指定日程的部分字段。
//
//	PUT /api/events/:id
//	请求体：{"title": "新标题", "color": "#FF6B6B"}  （只传要修改的字段）
//
// 使用 map[string]any 接收请求体，实现部分更新（PATCH 语义）。
// 不传的字段保持原值不变。
func (h *CalendarHandler) Update(c *gin.Context) {
	// 从 URL 路径参数中解析事件 ID
	// c.Param("id") 获取路径中的 :id 占位符的值
	// strconv.ParseUint 把字符串转为 uint64 数字
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint64("user_id")
	if err := h.svc.Update(id, userID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete 删除指定日程。
//
//	DELETE /api/events/:id
func (h *CalendarHandler) Delete(c *gin.Context) {
	// 用 _ 忽略错误（因为 id 格式不对时 ParseUint 返回 0，查询自然找不到记录）
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
