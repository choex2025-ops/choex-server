package agent

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/llm"
)

// AgentHandler AI 对话的 HTTP 处理器。
//
// 和其他 handler 不同的是，Agent 的响应不是普通 JSON，而是
// SSE（Server-Sent Events，服务端推送事件）流式响应。
//
// 什么是 SSE？
//
//	普通的 HTTP 请求：客户端发请求 → 服务端处理 → 一次性返回完整响应
//	SSE 流式请求：  客户端发请求 → 连接保持 → 服务端逐步推送数据 →
//	               全部推送完 → 连接关闭
//
//	SSE 的数据格式：
//	  data: {"content": "你"}
//	  data: {"content": "好"}
//	  data: {"done": true}
//
//	每条消息以 "data: " 开头，以 "\n\n" 结尾。
//	前端用 EventSource API 或 fetch + ReadableStream 来逐条读取。
//
// 为什么 AI 对话要用流式？
//	因为大模型是逐 token 生成回复的（一个字一个字地生成），
//	如果等全部生成完再返回，用户可能要等好几秒才能看到第一个字。
//	流式返回可以让用户看到 AI "正在输入"的效果，体验更好。
//	（类似 ChatGPT 逐字弹出的效果）
type AgentHandler struct {
	cfg *config.Config
}

// NewAgentHandler 创建 AI 对话处理器实例。
func NewAgentHandler(cfg *config.Config) *AgentHandler {
	return &AgentHandler{cfg: cfg}
}

// chatRequest AI 对话的请求体。
type chatRequest struct {
	Message string `json:"message" binding:"required"` // 用户发送的消息
}

// Chat 处理 AI 对话请求，以 SSE 流式返回 AI 的回复。
//
//	POST /api/agent/chat
//	请求体：{"message": "今天天气怎么样？"}
//
// 响应是 SSE 流，前端需要监听 data 事件：
//
//	data: {"content": "今"}
//	data: {"content": "天"}
//	data: {"content": "天"}
//	data: {"content": "气"}
//	...
//	data: {"done": true}
//
// 如果出错：
//
//	data: {"error": "AI 调用失败：..."}
func (h *AgentHandler) Chat(c *gin.Context) {
	// Step 1: 解析请求体
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Step 2: 设置 SSE 响应头
	// Content-Type: text/event-stream → 告诉浏览器这是 SSE 流
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	// Cache-Control: no-cache → 告诉浏览器不要缓存（流式数据不能缓存）
	c.Writer.Header().Set("Cache-Control", "no-cache")
	// Connection: keep-alive → 保持连接（支持长连接）
	c.Writer.Header().Set("Connection", "keep-alive")
	// 先写入状态码 200（必须在第一次 Write 之前调用 WriteHeader）
	c.Writer.WriteHeader(http.StatusOK)

	// Step 3: 获取 http.Flusher
	// Flusher 是什么？
	//   HTTP 响应通常会缓冲（攒够一定量再发送），Flusher 可以强制立即发送缓冲区数据。
	//   没有 Flusher，SSE 流式推送就无法实现——数据会一直积在缓冲区，直到连接关闭才发送。
	// c.Writer.(http.Flusher) 是 Go 的类型断言语法：
	//   如果 c.Writer 实现了 http.Flusher 接口，返回 (flusher, true)
	//   否则返回 (nil, false)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Step 4: 获取当前用户 ID（由 Auth 中间件注入）
	userID := c.GetUint64("user_id")

	// Step 5: 调用 Agent 处理逻辑
	// 返回一个 channel，通过它接收 AI 逐 token 生成的内容
	ch, err := processAgentChat(h.cfg, userID, []llm.Message{}, req.Message)
	if err != nil {
		// 初始化错误（如 API Key 缺失），以 SSE 格式返回
		data, _ := json.Marshal(gin.H{"error": err.Error()})
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
		return
	}

	// Step 6: 逐 token 推送给前端
	// for range ch 会持续接收 channel 中的数据，直到 channel 被关闭
	for res := range ch {
		// 如果收到错误，以 SSE 格式发送并结束
		if res.Err != nil {
			data, _ := json.Marshal(gin.H{"error": res.Err.Error()})
			c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
			return
		}
		// 发送当前 token 的内容
		// 例如 AI 生成了 "你"，JSON 编码后是 {"content": "你"}
		data, _ := json.Marshal(gin.H{"content": res.Content})
		// 按 SSE 格式写入：data: <json>\n\n
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		// 强制刷新缓冲区，立即发送给客户端
		flusher.Flush()
	}

	// Step 7: 通知前端流已结束
	data, _ := json.Marshal(gin.H{"done": true})
	c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
	flusher.Flush()
}
