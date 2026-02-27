package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"aiagent/internal/config"
	"aiagent/internal/logic"
	"aiagent/internal/svc"
	"aiagent/internal/types"
)

// Options 定义命令行对话模式的可选参数。
type Options struct {
	EnableTools  bool
	SessionID    string
	OperatorID   string
	TenantID     string
	SystemPrompt string
	MaxHistory   int
}

// RunChatREPL 启动命令行交互式对话。
func RunChatREPL(c config.Config, opt Options) error {
	sessionID := strings.TrimSpace(opt.SessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("cli-session-%d", time.Now().Unix())
	}

	maxHistory := opt.MaxHistory
	if maxHistory <= 0 {
		maxHistory = 12
	}

	operatorID := strings.TrimSpace(opt.OperatorID)
	if operatorID == "" {
		operatorID = "cli-user"
	}

	tenantID := strings.TrimSpace(opt.TenantID)
	if tenantID == "" {
		tenantID = "cli-tenant"
	}

	svcCtx := svc.NewServiceContext(c)
	chatLogic := logic.NewChatLogic(context.Background(), svcCtx)

	enableTools := opt.EnableTools
	history := make([]types.ChatMessage, 0, maxHistory*2)

	printBanner(sessionID, enableTools)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for {
		fmt.Print("\n你> ")
		if !scanner.Scan() {
			fmt.Println("\n会话结束。")
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			nextSessionID, nextTools, resetHistory, shouldExit := handleCommand(line, sessionID, enableTools, svcCtx)
			if shouldExit {
				fmt.Println("已退出 CLI 对话。")
				return nil
			}
			sessionID = nextSessionID
			enableTools = nextTools
			if resetHistory {
				history = history[:0]
			}
			continue
		}

		req := &types.ChatRequest{
			RequestId:    fmt.Sprintf("cli-%d", time.Now().UnixNano()),
			SessionId:    sessionID,
			EnableTools:  enableTools,
			OperatorId:   operatorID,
			TenantId:     tenantID,
			SystemPrompt: strings.TrimSpace(opt.SystemPrompt),
			Query:        line,
			Messages:     append([]types.ChatMessage(nil), history...),
		}

		resp, err := chatLogic.Chat(req)
		if err != nil {
			fmt.Printf("AI> 调用失败：%v\n", err)
			continue
		}

		fmt.Printf("AI> %s\n", strings.TrimSpace(resp.Reply))
		if enableTools {
			fmt.Printf("    [mode=%s intent=%s taskId=%s]\n", resp.Mode, resp.Intent, resp.TaskId)
		}

		history = append(history,
			types.ChatMessage{Role: "user", Content: line},
			types.ChatMessage{Role: "assistant", Content: resp.Reply},
		)
		history = trimHistory(history, maxHistory)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func printBanner(sessionID string, enableTools bool) {
	fmt.Println("----------------------------------------")
	fmt.Println("AIAgent CLI 对话模式已启动")
	fmt.Printf("sessionId: %s\n", sessionID)
	fmt.Printf("tools: %v\n", enableTools)
	fmt.Println("可用命令：")
	fmt.Println("  /help           查看帮助")
	fmt.Println("  /tools on|off   开关工具编排")
	fmt.Println("  /memory         查看当前 session 记忆")
	fmt.Println("  /new            新建 session（并清空历史）")
	fmt.Println("  /reset          清空当前历史")
	fmt.Println("  /quit           退出")
	fmt.Println("----------------------------------------")
}

func handleCommand(line, sessionID string, enableTools bool, svcCtx *svc.ServiceContext) (string, bool, bool, bool) {
	command := strings.TrimSpace(line)
	switch {
	case command == "/help":
		printBanner(sessionID, enableTools)
		return sessionID, enableTools, false, false
	case command == "/quit" || command == "/exit":
		return sessionID, enableTools, false, true
	case command == "/reset":
		fmt.Println("已清空本地对话历史。")
		return sessionID, enableTools, true, false
	case command == "/memory":
		printMemory(svcCtx, sessionID)
		return sessionID, enableTools, false, false
	case command == "/new":
		nextSession := fmt.Sprintf("cli-session-%d", time.Now().Unix())
		fmt.Printf("已切换到新 session: %s\n", nextSession)
		return nextSession, enableTools, true, false
	case command == "/tools on":
		fmt.Println("已开启工具编排模式。")
		return sessionID, true, false, false
	case command == "/tools off":
		fmt.Println("已关闭工具编排模式（仅大模型直聊）。")
		return sessionID, false, false, false
	default:
		fmt.Println("未知命令，输入 /help 查看帮助。")
		return sessionID, enableTools, false, false
	}
}

func printMemory(svcCtx *svc.ServiceContext, sessionID string) {
	session, ok := svcCtx.GetSession(sessionID)
	if !ok {
		fmt.Println("当前 session 暂无记忆。")
		return
	}

	memory := map[string]string{
		"shortCode": session.LastShortCode,
		"shortUrl":  session.LastShortURL,
		"longUrl":   session.LastLongURL,
		"expireAt":  session.LastExpireAt,
		"from":      session.LastFrom,
		"to":        session.LastTo,
		"intent":    session.LastIntent,
		"taskId":    session.LastTaskID,
	}
	raw, _ := json.MarshalIndent(memory, "", "  ")
	fmt.Printf("session memory:\n%s\n", string(raw))
}

func trimHistory(history []types.ChatMessage, maxRounds int) []types.ChatMessage {
	if maxRounds <= 0 {
		maxRounds = 12
	}
	maxMessages := maxRounds * 2
	if len(history) <= maxMessages {
		return history
	}
	start := len(history) - maxMessages
	return append([]types.ChatMessage(nil), history[start:]...)
}
