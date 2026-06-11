package agent

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/config"
)

type AgentHandler struct {
	svc *ChatService
}

func NewAgentHandler(cfg *config.Config) *AgentHandler {
	return &AgentHandler{svc: NewChatService(cfg)}
}

type chatRequest struct {
	Message string `json:"message" binding:"required"`
}

func (h *AgentHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	userID := c.GetUint64("user_id")

	ch, err := h.svc.SendMessage(c.Request.Context(), userID, req.Message)
	if err != nil {
		data, _ := json.Marshal(gin.H{"error": err.Error()})
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
		return
	}

	for res := range ch {
		if res.Err != nil {
			data, _ := json.Marshal(gin.H{"error": res.Err.Error()})
			c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
			return
		}
		data, _ := json.Marshal(gin.H{"content": res.Content})
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
	}

	data, _ := json.Marshal(gin.H{"done": true})
	c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
	flusher.Flush()
}
