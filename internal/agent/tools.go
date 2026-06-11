package agent

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/llm"
)

func toolCallSystemPrompt() string {
	return `你是 ChoexManager 个人生活管家。

【重要】区分以下两种回复方式：

1. 普通对话：当用户只是聊天、问候、自我介绍、问问题，直接回复文字。
   例如用户说"介绍一下你自己"→ 用自然语言介绍自己。

2. 工具调用：仅当用户明确要求操作数据时才使用工具。格式如下（严格 JSON，独占一行）：
   {"tool":"工具名","args":{...参数...}}

   例如用户说"帮我记一笔午餐30元"→ 调用 create_bill 工具。

可用工具：
- list_events：列出日程列表（无参数）
- create_event：创建日程。参数: title, start_time, end_time, description(可选), location(可选)
- list_bills：列出记账记录（无参数）
- create_bill：创建记账。参数: amount(数字), type("expense"或"income"), category(可选), note(可选)`
}

var validTools = map[string]bool{
	"list_events":  true,
	"create_event": true,
	"list_bills":   true,
	"create_bill":  true,
}

func executeTool(name string, userID uint64, jsonArgs string) string {
	if !validTools[name] {
		return `{"error": "unknown tool: ` + name + `"}`
	}

	var args map[string]any
	json.Unmarshal([]byte(jsonArgs), &args)

	switch name {
	case "list_events":
		var events []model.Event
		database.DB.Where("user_id = ?", userID).Order("start_time ASC").Find(&events)
		b, _ := json.Marshal(events)
		return string(b)

	case "create_event":
		title, okT := args["title"].(string)
		startStr, okS := args["start_time"].(string)
		endStr, _ := args["end_time"].(string)
		if !okT || title == "" || !okS || startStr == "" {
			return `{"error": "create_event requires: title, start_time"}`
		}
		desc, _ := args["description"].(string)
		loc, _ := args["location"].(string)

		startTime, _ := time.Parse(time.RFC3339, startStr)
		endTime, _ := time.Parse(time.RFC3339, endStr)

		event := model.Event{
			UserID:      userID,
			Title:       title,
			StartTime:   startTime,
			EndTime:     endTime,
			Description: desc,
			Location:    loc,
		}
		database.DB.Create(&event)
		return `{"message": "日程已创建: ` + title + `"}`

	case "list_bills":
		var bills []model.Bill
		database.DB.Where("user_id = ?", userID).Order("bill_date DESC").Limit(20).Find(&bills)
		b, _ := json.Marshal(bills)
		return string(b)

	case "create_bill":
		amount, okAmt := args["amount"].(float64)
		billType, okType := args["type"].(string)
		if !okAmt || amount <= 0 || !okType || (billType != "expense" && billType != "income") {
			return `{"error": "create_bill requires: amount(>0), type(expense or income)"}`
		}
		category, _ := args["category"].(string)
		note, _ := args["note"].(string)

		bill := model.Bill{
			UserID:   userID,
			Amount:   amount,
			Type:     billType,
			Category: category,
			Note:     note,
			BillDate: time.Now().Format("2006-01-02"),
		}
		database.DB.Create(&bill)
		return `{"message": "已记录: ` + billType + ` ¥` + formatFloatStr(amount) + `"}`

	default:
		return `{"error": "unknown tool"}`
	}
}

func formatFloatStr(f float64) string {
	s := time.Time{}.Format("") // unused, placeholder
	_ = s
	b, _ := json.Marshal(f)
	return string(b)
}

func processAgentChat(cfg *config.Config, userID uint64, history []llm.Message, userMsg string) (<-chan llm.StreamResult, error) {
	// First: call LLM with tool awareness (non-streaming to detect tool calls)
	systemPrompt := toolCallSystemPrompt()

	content, _, err := llm.CallDeepSeek(systemPrompt, history, false)
	if err != nil {
		return nil, err
	}

	// Check if LLM wants to call a tool (must be clean JSON with known tool name)
	content = trimSpace(content)
	var toolReq struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}
	if len(content) > 0 && content[0] == '{' &&
		json.Unmarshal([]byte(content), &toolReq) == nil &&
		toolReq.Tool != "" && validTools[toolReq.Tool] {
		argsJSON, _ := json.Marshal(toolReq.Args)
		result := executeTool(toolReq.Tool, userID, string(argsJSON))

		// Send result back to LLM for final response (streaming)
		newHistory := append(history,
			llm.Message{Role: "user", Content: userMsg},
			llm.Message{Role: "assistant", Content: "正在调用工具 " + toolReq.Tool + "..."},
			llm.Message{Role: "user", Content: "工具返回结果: " + result + "\n请用自然语言向用户解释这个结果。"},
		)

		_, ch, err := llm.CallDeepSeek(systemPrompt, newHistory, true)
		return ch, err
	}

	// No tool call: stream the response to user
	ch := make(chan llm.StreamResult, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamResult{Content: content}
	}()

	return ch, nil
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
