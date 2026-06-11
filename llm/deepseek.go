// Package llm 封装了对大语言模型（LLM）API 的调用。
//
// 当前适配的模型：DeepSeek（https://platform.deepseek.com）
//
// DeepSeek 是一家中国 AI 公司，提供兼容 OpenAI API 格式的接口。
// 因此本包使用 OpenAI 官方的 Go SDK（github.com/openai/openai-go）来调用 DeepSeek，
// 只需要把 BaseURL 指向 DeepSeek 的 API 地址即可。
//
// 核心功能：
//   - CallDeepSeek：统一入口，支持流式和非流式两种调用模式
//   - callNonStreaming：一次性获取完整回复
//   - callStreaming：通过 channel 逐 token 推送（适合实时展示）
//
// 使用示例（流式）：
//
//	_, ch, err := CallDeepSeek("你是一个助手", history, true)
//	for res := range ch {
//	    if res.Err != nil { ... }
//	    fmt.Print(res.Content) // 逐个 token 打印
//	}
//
// 使用示例（非流式）：
//
//	content, _, err := CallDeepSeek("你是一个助手", history, false)
//	fmt.Println(content) // 打印完整回复
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
//
// 这是一个包级私有函数（小写开头），只在 llm 包内部使用。
// 和 config 包中的 getEnv 功能相同，但独立存在以避免循环依赖。
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// StreamResult 流式响应的单个结果块。
//
// 流式返回时，每生成一个 token，channel 就会收到一个 StreamResult。
// 要么 Content 有值（正常的 token 文本），要么 Err 有值（出错了）。
type StreamResult struct {
	Content string // 本次收到的 token 文本，如 "你"、"好"
	Err     error  // 错误（非 nil 时表示流出错终止，Content 为空）
}

// Message 表示一条对话消息，用于构建 Chat Completions 请求的消息数组。
//
// 和 OpenAI 的消息格式对齐：
//   - Role: "system"（系统提示词）、"user"（用户消息）、"assistant"（AI 回复）
//   - Content: 消息正文
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// newClientFn 创建 OpenAI 客户端的函数。
//
// 这是一个包级变量（不是常量），目的是在测试中可以替换为指向
// httptest 模拟服务器的客户端，实现不依赖外部 API 的单元测试。
//
// 生产环境中，它创建一个指向 DeepSeek API 的客户端。
// 测试环境中，通过修改这个变量指向本地测试服务器。
var newClientFn = func(apiKey, baseURL string) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(apiKey),     // 设置 API Key（DeepSeek 的 API Key）
		option.WithBaseURL(baseURL),   // 设置 API 地址（DeepSeek 的 /v1 端点）
	)
}

// CallDeepSeek 发送对话请求到 DeepSeek API。
//
// 这是本包对外暴露的唯一入口函数，统一处理流式和非流式两种调用模式。
//
// 参数说明：
//   - systemPrompt：系统提示词（定义 AI 的角色和行为），可以为空字符串
//   - history：历史对话消息列表，按 user/assistant 交替排列
//     每次请求会把完整历史发送给 API，以保持多轮对话的上下文
//     注意：不要把消息累积太多，否则 token 消耗会快速增长
//   - stream：
//     false → 阻塞等待完整回复，返回 (完整内容, nil, nil)
//     true  → 立即返回 ("", channel, nil)。调用方通过 for range 逐 token 接收
//
// 返回值：
//   - string：非流式模式下的完整回复（流式模式下为空字符串）
//   - <-chan StreamResult：流式模式下的 token 通道（非流式模式下为 nil）
//   - error：初始化错误（如 API Key 缺失、参数错误）
//
// 流式返回的 channel 使用约定：
//
//	for res := range ch {
//	    if res.Err != nil {
//	        // 流处理出错，结束
//	    }
//	    fmt.Print(res.Content) // 打印当前 token
//	}
//	// channel 关闭，流结束
//
// 所需环境变量：
//   - DEEPSEEK_API_KEY（必填）：DeepSeek 平台 API Key
//   - DEEPSEEK_BASE_URL（可选）：API 地址，默认 https://api.deepseek.com/
//   - DEEPSEEK_MODEL（可选）：模型名称，默认 deepseek-v4-pro
func CallDeepSeek(systemPrompt string, history []Message, stream bool) (string, <-chan StreamResult, error) {
	// ---- 读取配置 ----
	apiKey := getEnv("DEEPSEEK_API_KEY", "")
	if apiKey == "" {
		return "", nil, fmt.Errorf("请设置 DEEPSEEK_API_KEY 环境变量")
	}

	baseURL := getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/")
	model := getEnv("DEEPSEEK_MODEL", "deepseek-v4-pro")

	// ---- 创建客户端（连接指向 DeepSeek API） ----
	client := newClientFn(apiKey, baseURL)

	// ---- 构建消息数组 ----
	// 消息数组的结构：[system(可选), user, assistant, user, assistant, ...]
	// 这是 OpenAI Chat Completions API 的标准格式
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+1)

	// 如果有系统提示词，放在第一条
	if systemPrompt != "" {
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	// 按顺序追加历史对话
	for _, msg := range history {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
		// 注意：跳过 system 角色的历史消息
		// 因为 system prompt 只在开头设置一次，不会出现在对话历史中
	}

	// ---- 构建 API 请求参数 ----
	params := openai.ChatCompletionNewParams{
		Model:    model,    // 使用哪个模型（deepseek-v4-pro）
		Messages: messages, // 消息数组
	}

	// ---- 根据 stream 参数选择调用模式 ----
	if stream {
		// 流式模式：创建 channel，启动后台 goroutine 推 token
		ch := make(chan StreamResult)
		go callStreaming(client, context.Background(), params, ch)
		return "", ch, nil
	}

	// 非流式模式：同步等待完整回复
	content, err := callNonStreaming(client, context.Background(), params)
	return content, nil, err
}

// callNonStreaming 一次性请求，返回完整回复。
//
// 工作原理：
//  1. 发送 HTTP POST 请求到 DeepSeek API
//  2. DeepSeek 生成完整回复（可能需要几秒到几十秒）
//  3. 返回包含完整 JSON 的 HTTP 响应
//  4. SDK 解析 JSON，提取 choices[0].message.content
//
// 适合场景：后台批量处理、不需要实时展示的场景
//
// 参数：
//   - client：OpenAI 客户端实例
//   - ctx：上下文（用于超时控制和取消）
//   - params：请求参数
//
// 返回：AI 的完整回复文本 和 可能的错误
func callNonStreaming(client openai.Client, ctx context.Context, params openai.ChatCompletionNewParams) (string, error) {
	// client.Chat.Completions.New 发送同步请求
	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("API 调用失败: %w", err)
	}

	// 拼接所有 choices 的回复内容
	// 通常只有一个 choice，但 API 可能返回多个（n > 1 时）
	var sb strings.Builder
	for _, choice := range resp.Choices {
		sb.WriteString(choice.Message.Content)
	}
	return sb.String(), nil
}

// callStreaming 流式请求，通过 channel 逐 token 发送给调用方。
//
// 流式输出原理（SSE，Server-Sent Events）：
//
//	非流式：HTTP 请求 → 服务端生成完整回复 → JSON 响应 → 连接关闭
//
//	流式：  HTTP 请求(stream=true) → 连接保持打开 →
//	        服务端每生成一个 token 就推送一条 SSE chunk →
//	        SDK 逐行读取 response body →
//	        调用方的 for range ch 接收到每个 token →
//	        全部 token 生成完 → 服务端发送 [DONE] → 连接关闭 →
//	        goroutine 关闭 channel
//
//	SSE 原始数据格式（HTTP response body）：
//	  data: {"choices":[{"delta":{"content":"你"}}]}
//	  data: {"choices":[{"delta":{"content":"好"}}]}
//	  data: [DONE]
//
//	一句话总结：LLM 天然逐 token 生成，流式就是不等攒齐，生成一个推一个。
//
// 参数：
//   - client：OpenAI 客户端实例
//   - ctx：上下文
//   - params：请求参数（stream 会自动设为 true）
//   - ch：发送 StreamResult 的 channel（函数结束后关闭）
func callStreaming(client openai.Client, ctx context.Context, params openai.ChatCompletionNewParams, ch chan<- StreamResult) {
	// defer close(ch) 确保无论函数如何退出，channel 最终都会被关闭
	// 调用方的 for range 在 channel 关闭后自动退出循环
	defer close(ch)

	// NewStreaming 发送流式请求，返回一个流迭代器
	stream := client.Chat.Completions.NewStreaming(ctx, params)

	// stream.Next() 阻塞等待下一个 token chunk
	// 每次返回 true 表示有新数据可读
	// 流结束或出错时返回 false
	for stream.Next() {
		// stream.Current() 获取当前 chunk
		chunk := stream.Current()
		// Chunk 中的 Choices 是增量（delta），不是完整响应
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				// 把这个 token 推送给调用方
				ch <- StreamResult{Content: delta.Content}
			}
		}
	}

	// stream.Err() 返回流处理过程中的错误（如果有）
	// 正常的流结束（收到 [DONE]）不会导致 Err 非 nil
	if err := stream.Err(); err != nil {
		// 发送最后一个带错误的 StreamResult，然后返回（defer close(ch) 执行）
		ch <- StreamResult{Err: fmt.Errorf("流式调用失败: %w", err)}
	}
}
