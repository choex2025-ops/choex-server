package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// getEnv 读取环境变量，不存在时返回默认值。
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// StreamResult 流式响应的单个结果块。
type StreamResult struct {
	Content string // 本次收到的 token 文本
	Err     error  // 错误（非 nil 时表示流结束，Content 为空）
}

// Message 表示一条对话消息，用于构建 Chat Completions 请求的消息数组。
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// newClientFn 创建 openai 客户端。在测试中可替换以指向 httptest server。
var newClientFn = func(apiKey, baseURL string) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
}

// CallDeepSeek 发送对话请求。
//
//	systemPrompt: 系统提示词（可为空），作为第一条 system 消息发送。
//	history:      对话历史，按顺序包含 user/assistant 消息。
//	              每次请求会将完整历史发送给 API，保持多轮对话的上下文。
//	stream=false: 阻塞等待完整回复，返回 (content, nil, nil)。
//	stream=true:  立即返回 ("", ch, nil)。调用方通过 for range ch 逐 token 接收：
//	                  ch <- StreamResult{Content: "你"}
//	                  ch <- StreamResult{Content: "好"}
//	                  ch <- StreamResult{Err: ...} 或 close(ch) 表示结束
//
// 对应重构.md 第1~2步。
func CallDeepSeek(systemPrompt string, history []Message, stream bool) (string, <-chan StreamResult, error) {
	apiKey := getEnv("DEEPSEEK_API_KEY", "")
	if apiKey == "" {
		return "", nil, fmt.Errorf("请设置 DEEPSEEK_API_KEY 环境变量")
	}

	baseURL := getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/")
	model := getEnv("DEEPSEEK_MODEL", "deepseek-v4-pro")

	client := newClientFn(apiKey, baseURL)

	// 构建消息数组：system(可选) + 对话历史(user/assistant 交替)
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+1)

	if systemPrompt != "" {
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	for _, msg := range history {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
		// system 角色的历史消息忽略，不在多轮对话中重复发送
	}

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	if stream {
		ch := make(chan StreamResult)
		go callStreaming(client, context.Background(), params, ch)
		return "", ch, nil
	}

	content, err := callNonStreaming(client, context.Background(), params)
	return content, nil, err
}

// callNonStreaming 一次性请求，返回完整回复。
func callNonStreaming(client openai.Client, ctx context.Context, params openai.ChatCompletionNewParams) (string, error) {
	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("API 调用失败: %w", err)
	}

	var sb strings.Builder
	for _, choice := range resp.Choices {
		sb.WriteString(choice.Message.Content)
	}
	return sb.String(), nil
}

// callStreaming 流式请求，通过 channel 逐 token 发送给调用方。
//
//	启动 goroutine 后台读取 SSE 流，每收到一个 token 就 send 到 ch。
//	读取完毕后 close(ch) 通知调用方结束。
//	如果中途出错，发送最后一个带的 StreamResult{Err: ...} 然后 close。
//
// 流式输出原理（SSE，Server-Sent Events）：
//	非流式：HTTP 请求 → 服务端生成完整回复 → JSON 响应 → 连接关闭
//	流式：  HTTP 请求(stream=true) → 连接保持 → 每生成 token 就推送 SSE chunk →
//	        SDK 逐行读取 response body → Next() 每读一个 chunk 返回 true →
//	        全部 token 生成完发送 [DONE] → 连接关闭
//
//	SSE 数据格式（HTTP response body）：
//	  data: {"choices":[{"delta":{"content":"你"}}]}
//	  data: {"choices":[{"delta":{"content":"好"}}]}
//	  data: [DONE]
//
//	一句话：LLM 天然逐 token 生成，服务端不等攒齐就推送。
func callStreaming(client openai.Client, ctx context.Context, params openai.ChatCompletionNewParams, ch chan<- StreamResult) {
	defer close(ch)

	stream := client.Chat.Completions.NewStreaming(ctx, params)

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				ch <- StreamResult{Content: delta.Content}
			}
		}
	}

	if err := stream.Err(); err != nil {
		ch <- StreamResult{Err: fmt.Errorf("流式调用失败: %w", err)}
	}
}
