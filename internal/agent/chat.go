package agent

import (
	"context"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/llm"
)

type ChatService struct {
	cfg *config.Config
}

func NewChatService(cfg *config.Config) *ChatService {
	return &ChatService{cfg: cfg}
}

type StreamChunk struct {
	Content string
	Err     error
}

func (s *ChatService) SendMessage(ctx context.Context, userID uint64, message string) (<-chan StreamChunk, error) {
	systemPrompt := "你是 ChoexManager 的个人生活管家。你可以帮助用户管理日程、记账、查询密码簿。请用温和、简洁的语言回复。"

	_, rawCh, err := llm.CallDeepSeek(systemPrompt, []llm.Message{
		{Role: "user", Content: message},
	}, true)

	if err != nil {
		return nil, err
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		for res := range rawCh {
			ch <- StreamChunk{Content: res.Content, Err: res.Err}
		}
	}()

	return ch, nil
}
