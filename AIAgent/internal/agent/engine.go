package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"aiagent/internal/svc"
	"aiagent/internal/tools"
	"aiagent/internal/types"
)

type ExecuteResult struct {
	TaskID  string           // 任务唯一 ID
	TraceID string           // 链路追踪 ID
	Intent  string           // 识别到的意图
	Plan    []string         // 执行计划步骤
	Calls   []types.ToolCall // 工具调用记录
	Answer  string           // 最终回答
	Status  string           // 执行状态
}

// Engine 抽象编排引擎接口，rule 与 eino 都实现此接口。
type Engine interface {
	Execute(ctx context.Context, req *types.ExecuteRequest) (*ExecuteResult, error)
}

// NewEngine 根据配置创建具体引擎实现。
func NewEngine(svcCtx *svc.ServiceContext) Engine {
	engine := strings.ToLower(strings.TrimSpace(svcCtx.Config.Orchestration.Engine))
	switch engine {
	case "eino":
		return NewEinoEngine(svcCtx)
	default:
		return NewRuleEngine(svcCtx)
	}
}

type RuleEngine struct {
	svcCtx *svc.ServiceContext
	tools  *tools.InternalClient
}

// NewRuleEngine 创建规则引擎（兜底与本地联调用）。
func NewRuleEngine(svcCtx *svc.ServiceContext) *RuleEngine {
	return &RuleEngine{
		svcCtx: svcCtx,
		tools:  tools.NewInternalClient(svcCtx),
	}
}

// Execute 是规则引擎主流程：意图识别 -> 计划 -> 工具执行 -> 汇总。
// 注意：意图识别统一走大模型，不再使用关键词规则匹配。
func (e *RuleEngine) Execute(ctx context.Context, req *types.ExecuteRequest) (*ExecuteResult, error) {
	if strings.TrimSpace(req.RequestId) == "" {
		return nil, errors.New("requestId is required")
	}
	if strings.TrimSpace(req.OperatorId) == "" {
		return nil, errors.New("operatorId is required")
	}
	if strings.TrimSpace(req.TenantId) == "" {
		return nil, errors.New("tenantId is required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("query is required")
	}

	taskID := newID("task")
	traceID := newID("trace")

	intent, intentErr := inferIntentByLLM(ctx, e.svcCtx, req.Query)
	if intentErr != nil {
		toolCalls := []types.ToolCall{{
			Tool:   "intent.llm",
			Input:  req.Query,
			Status: "error",
			Error:  intentErr.Error(),
		}}

		return &ExecuteResult{
			TaskID:  taskID,
			TraceID: traceID,
			Intent:  "unknown",
			Plan:    buildPlan("unknown"),
			Calls:   toolCalls,
			Answer:  summarize("unknown", toolCalls),
			Status:  finalStatus(toolCalls),
		}, nil
	}

	intent = normalizeIntent(intent)
	plan := buildPlan(intent)
	toolCalls := make([]types.ToolCall, 0, 2)

	switch intent {
	case "create_shortlink":
		toolCalls = append(toolCalls, e.tools.CreateShortLink(ctx, req))
	case "update_shortlink":
		if !e.svcCtx.Config.Orchestration.AllowWriteActions {
			toolCalls = append(toolCalls, types.ToolCall{Tool: "shortener.update", Input: req.Query, Status: "blocked", Error: "write actions disabled"})
		} else {
			toolCalls = append(toolCalls, e.tools.UpdateShortLink(ctx, req))
		}
	case "query_traffic":
		toolCalls = append(toolCalls, e.tools.QueryTrafficSummary(ctx, req))
	case "update_and_query":
		if e.svcCtx.Config.Orchestration.AllowWriteActions {
			toolCalls = append(toolCalls, e.tools.UpdateShortLink(ctx, req))
		} else {
			toolCalls = append(toolCalls, types.ToolCall{Tool: "shortener.update", Input: req.Query, Status: "blocked", Error: "write actions disabled"})
		}
		toolCalls = append(toolCalls, e.tools.QueryTrafficSummary(ctx, req))
	default:
		toolCalls = append(toolCalls, types.ToolCall{Tool: "intent.fallback", Input: req.Query, Status: "unknown", Error: "unsupported intent"})
	}

	return &ExecuteResult{
		TaskID:  taskID,
		TraceID: traceID,
		Intent:  intent,
		Plan:    plan,
		Calls:   toolCalls,
		Answer:  summarize(intent, toolCalls),
		Status:  finalStatus(toolCalls),
	}, nil
}

// buildPlan 根据意图生成默认执行步骤。
func buildPlan(intent string) []string {
	switch intent {
	case "create_shortlink":
		return []string{"intent analyze", "plan create", "call shortener.create", "verify result"}
	case "update_shortlink":
		return []string{"intent analyze", "plan update", "call shortener.update", "verify result"}
	case "query_traffic":
		return []string{"intent analyze", "plan query", "call traffic.summary", "verify result"}
	case "update_and_query":
		return []string{"intent analyze", "plan update+query", "call shortener.update", "call traffic.summary", "verify result"}
	default:
		return []string{"intent analyze", "unsupported fallback"}
	}
}

// summarize 汇总工具调用结果，输出简要说明。
func summarize(intent string, calls []types.ToolCall) string {
	if len(calls) == 0 {
		return "未执行任何工具调用"
	}

	ok := 0
	fail := 0
	for _, call := range calls {
		if call.Status == "ok" {
			ok++
		} else {
			fail++
		}
	}

	return fmt.Sprintf("意图: %s；工具调用成功 %d 次，失败 %d 次", intent, ok, fail)
}

// finalStatus 根据调用状态计算任务最终状态。
func finalStatus(calls []types.ToolCall) string {
	if len(calls) == 0 {
		return "no_op"
	}
	for _, call := range calls {
		if call.Status != "ok" {
			return "partial_failed"
		}
	}
	return "succeeded"
}

// newID 生成简单随机 ID。
func newID(prefix string) string {
	raw := make([]byte, 8)
	_, _ = rand.Read(raw)
	return prefix + "-" + hex.EncodeToString(raw)
}
