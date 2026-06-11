package agent

import (
	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/llm"
)

// processAgentChat handles agent conversation with pure chat mode.
// Tool calling will be re-implemented using proper function calling API in the future.
func processAgentChat(cfg *config.Config, userID uint64, history []llm.Message, userMsg string) (<-chan llm.StreamResult, error) {
	systemPrompt := "你是 ChoexManager 个人生活管家，可以用中文和用户交流。如果用户询问日程、记账、密码等数据，可以提示用户去对应的应用查看。"

	allHistory := append(history, llm.Message{Role: "user", Content: userMsg})

	_, ch, err := llm.CallDeepSeek(systemPrompt, allHistory, true)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
