// Package agent 负责 AI 智能体（Agent）的对话逻辑。
//
// 什么是 Agent（智能体）？
//
//	传统聊天机器人：用户发消息 → AI 回复文本
//	智能体：用户发消息 → AI 可以调用工具（查日程、记账等）→ 根据工具结果回复
//	       相当于 AI 有了"手"，不只是聊天，还能帮用户做事
//
// 当前版本的 Agent 实现：
//	目前只支持纯聊天模式，AI 用中文和用户交流。
//	如果用户问到日程、记账等数据，AI 会提示用户去对应的功能页面查看。
//
// 未来计划：
//	实现真正的 Tool Calling（工具调用），让 AI 可以直接帮用户：
//	  - 查询今日日程
//	  - 创建新的账单
//	  - 搜索密码记录
//	  等等（通过调用对应的内部 API）
package agent

import (
	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/llm"
)

// processAgentChat 处理智能体对话的核心逻辑。
//
// 当前版本（纯聊天模式）：
//  1. 设置一个中文系统提示词（system prompt），告诉 AI 它的角色
//  2. 把用户的消息和历史对话一起发给 DeepSeek
//  3. 通过 channel 流式返回 AI 的回复
//
// 参数：
//   - cfg：应用配置（包含 DeepSeek API Key）
//   - userID：当前用户 ID（预留，未来用于工具调用的权限控制）
//   - history：历史对话消息（user/assistant 交替）
//   - userMsg：用户最新发送的消息
//
// 返回：流式结果 channel 和 可能的错误
//
// 未来 Tool Calling 版本的设计思路：
//
//	1. 定义工具列表（JSON Schema 格式），告诉 AI 有哪些工具可用
//	2. 发送用户消息时附带工具定义
//	3. AI 返回"要调用的工具名+参数"而不是"文本回复"
//	4. 代码执行工具，把执行结果再发给 AI
//	5. AI 根据工具结果生成最终回复
func processAgentChat(cfg *config.Config, userID uint64, history []llm.Message, userMsg string) (<-chan llm.StreamResult, error) {
	// 系统提示词：定义 AI 的角色和行为
	// 这个提示词告诉 AI：
	//   - 它是"ChoexManager 个人生活管家"
	//   - 可以用中文交流
	//   - 如果用户询问日程/记账/密码等数据，引导用户去对应功能查看（因为没有工具调用能力）
	systemPrompt := "你是 ChoexManager 个人生活管家，可以用中文和用户交流。如果用户询问日程、记账、密码等数据，可以提示用户去对应的应用查看。"

	// 构建完整的消息数组：system prompt + 历史对话包含最新用户消息
	// 注意：历史消息中已经包含了用户最新消息（由调用方 append 后传入）
	allHistory := append(history, llm.Message{Role: "user", Content: userMsg})

	// 调用 DeepSeek API，stream=true 开启流式返回
	// llm.CallDeepSeek 返回三个值：
	//   - 完整内容（stream=false 时才有值，这里忽略）
	//   - 流式 channel（stream=true 时使用，逐 token 接收）
	//   - 错误（API Key 不存在等初始化错误）
	_, ch, err := llm.CallDeepSeek(systemPrompt, allHistory, true)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
