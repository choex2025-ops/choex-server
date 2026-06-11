package agent

import (
	"encoding/json"
	"os"
	"time"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/llm"
)

func toolCallSystemPrompt() string {
	return `你是 ChoexManager 个人生活管家。你可以使用以下工具帮助用户：

可用工具：
1. list_events - 列出用户的日程列表
2. create_event - 创建新日程。参数: title(标题), start_time(开始时间), end_time(结束时间), description(描述), location(地点)
3. list_bills - 列出用户的记账记录
4. create_bill - 创建记账记录。参数: amount(金额), type(expense/income), category(餐饮/交通/购物/娱乐/其他), note(备注)

当你需要调用工具时，以以下JSON格式回复，不要有任何其他内容：
{"tool":"工具名","args":{...参数...}}

收到工具结果后，用自然语言向用户解释结果。`
}

func executeTool(name string, userID uint64, jsonArgs string) string {
	var args map[string]any
	json.Unmarshal([]byte(jsonArgs), &args)

	switch name {
	case "list_events":
		var events []model.Event
		database.DB.Where("user_id = ?", userID).Order("start_time ASC").Find(&events)
		b, _ := json.Marshal(events)
		return string(b)

	case "create_event":
		title, _ := args["title"].(string)
		startStr, _ := args["start_time"].(string)
		endStr, _ := args["end_time"].(string)
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
		amount, _ := args["amount"].(float64)
		billType, _ := args["type"].(string)
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

	// Check if LLM wants to call a tool
	var toolReq struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}
	if json.Unmarshal([]byte(content), &toolReq) == nil && toolReq.Tool != "" {
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

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
