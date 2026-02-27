package agent

import (
	"context"
	"errors"
	"strings"
	"time"

	"aiagent/internal/svc"

	einoOpenAI "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

type llmIntentOutput struct {
	Intent string `json:"intent"`
}

// normalizeIntent 约束意图值到系统支持的枚举。
func normalizeIntent(intent string) string {
	switch strings.TrimSpace(strings.ToLower(intent)) {
	case "create_shortlink":
		return "create_shortlink"
	case "update_shortlink":
		return "update_shortlink"
	case "query_traffic":
		return "query_traffic"
	case "update_and_query":
		return "update_and_query"
	default:
		return "unknown"
	}
}

// inferIntentByLLM 调用大模型完成意图识别，不走规则关键词匹配。
func inferIntentByLLM(ctx context.Context, svcCtx *svc.ServiceContext, query string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "unknown", errors.New("query is empty")
	}

	conf := svcCtx.Config.LLM
	if strings.TrimSpace(conf.Model) == "" {
		return "unknown", errors.New("LLM.Model is empty")
	}

	timeoutMs := conf.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 8000
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	apiKey := conf.APIKey
	if strings.TrimSpace(apiKey) == "" {
		apiKey = "EMPTY_KEY"
	}

	chatModel, err := einoOpenAI.NewChatModel(runCtx, &einoOpenAI.ChatModelConfig{
		APIKey:  apiKey,
		Model:   conf.Model,
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		return "unknown", err
	}

	system := `You are intent-agent.
Return ONLY JSON with field "intent".
Allowed values: create_shortlink, update_shortlink, query_traffic, update_and_query, unknown.
If user asks to change expiration/expiry/有效期/过期时间, classify as update_shortlink.`

	resp, err := chatModel.Generate(runCtx, []*schema.Message{
		schema.SystemMessage(system),
		schema.UserMessage(q),
	})
	if err != nil {
		return "unknown", err
	}

	var out llmIntentOutput
	if err := decodeJSONContent(resp.Content, &out); err != nil {
		return "unknown", err
	}

	return normalizeIntent(out.Intent), nil
}
