package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aiagent/internal/svc"
	"aiagent/internal/tools"
	"aiagent/internal/types"

	einoOpenAI "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type EinoEngine struct {
	svcCtx       *svc.ServiceContext
	ruleFallback *RuleEngine
	tools        *tools.InternalClient
}

// dagState 是 Eino 工作流中在各节点间流转的状态对象。
type dagState struct {
	Req     *types.ExecuteRequest
	Intent  string
	Planner plannerDecision
	Calls   []types.ToolCall
	Answer  string
}

type intentDecision struct {
	Intent string `json:"intent"`
}

type plannerDecision struct {
	Intent  string   `json:"intent"`
	Steps   []string `json:"steps"`
	Summary string   `json:"summary"`
}

type verifierDecision struct {
	Answer string `json:"answer"`
}

func NewEinoEngine(svcCtx *svc.ServiceContext) *EinoEngine {
	return &EinoEngine{
		svcCtx:       svcCtx,
		ruleFallback: NewRuleEngine(svcCtx),
		tools:        tools.NewInternalClient(svcCtx),
	}
}

// Execute 是 Eino 引擎入口：构建模型、运行 DAG、汇总结果并返回。
func (e *EinoEngine) Execute(ctx context.Context, req *types.ExecuteRequest) (*ExecuteResult, error) {
	if err := validateExecuteRequest(req); err != nil {
		return nil, err
	}

	chatModel, err := e.newChatModel(ctx)
	if err != nil {
		return e.fallbackOrErr(ctx, req, fmt.Errorf("build eino chat model failed: %w", err))
	}

	state := &dagState{Req: req}
	out, err := e.runDAG(ctx, chatModel, state)
	if err != nil {
		return e.fallbackOrErr(ctx, req, fmt.Errorf("eino dag failed: %w", err))
	}

	intent := strings.TrimSpace(out.Planner.Intent)
	if intent == "" {
		intent = strings.TrimSpace(out.Intent)
	}
	intent = normalizeIntent(intent)
	if intent == "unknown" {
		if inferred, inferErr := e.intentAgent(ctx, chatModel, req); inferErr == nil {
			intent = normalizeIntent(inferred)
		}
	}

	steps := nonEmptySteps(out.Planner.Steps, buildPlan(intent))
	calls := out.Calls
	answer := strings.TrimSpace(out.Answer)
	if answer == "" {
		answer = summarize(intent, calls)
	}

	return &ExecuteResult{
		TaskID:  newID("task"),
		TraceID: newID("trace"),
		Intent:  intent,
		Plan:    steps,
		Calls:   calls,
		Answer:  answer,
		Status:  finalStatus(calls),
	}, nil
}

// fallbackOrErr 控制是否在 Eino 失败时降级到 rule 引擎。
func (e *EinoEngine) fallbackOrErr(ctx context.Context, req *types.ExecuteRequest, rootErr error) (*ExecuteResult, error) {
	if !e.svcCtx.Config.Orchestration.FallbackToRule {
		return nil, rootErr
	}
	return e.ruleFallback.Execute(ctx, req)
}

// newChatModel 创建 Eino OpenAI ChatModel 客户端。
func (e *EinoEngine) newChatModel(ctx context.Context) (*einoOpenAI.ChatModel, error) {
	conf := e.svcCtx.Config.LLM
	if strings.TrimSpace(conf.Model) == "" {
		return nil, errors.New("LLM.Model is empty")
	}

	apiKey := conf.APIKey
	if strings.TrimSpace(apiKey) == "" {
		apiKey = "EMPTY_KEY"
	}

	return einoOpenAI.NewChatModel(ctx, &einoOpenAI.ChatModelConfig{
		APIKey:  apiKey,
		Model:   conf.Model,
		BaseURL: conf.BaseURL,
	})
}

// runDAG 构建并执行 compose Workflow：intent -> planner -> tool -> verifier。
func (e *EinoEngine) runDAG(ctx context.Context, chatModel *einoOpenAI.ChatModel, input *dagState) (*dagState, error) {
	wf := compose.NewWorkflow[*dagState, *dagState]()

	intentNode := wf.AddLambdaNode("intent-agent", compose.InvokableLambda(func(c context.Context, in *dagState) (*dagState, error) {
		intent, err := e.intentAgent(c, chatModel, in.Req)
		if err != nil {
			return nil, err
		}
		in.Intent = intent
		return in, nil
	}))
	intentNode.AddInput(compose.START)

	plannerNode := wf.AddLambdaNode("planner-agent", compose.InvokableLambda(func(c context.Context, in *dagState) (*dagState, error) {
		planner, err := e.plannerAgent(c, chatModel, in.Req, in.Intent)
		if err != nil {
			return nil, err
		}
		in.Planner = *planner
		if strings.TrimSpace(in.Planner.Intent) == "" {
			in.Planner.Intent = in.Intent
		}
		return in, nil
	}))
	plannerNode.AddInput("intent-agent")

	toolNode := wf.AddLambdaNode("tool-agent", compose.InvokableLambda(func(c context.Context, in *dagState) (*dagState, error) {
		calls, err := e.toolAgent(c, chatModel, in)
		if err != nil {
			return nil, err
		}
		in.Calls = calls
		return in, nil
	}))
	toolNode.AddInput("planner-agent")

	verifierNode := wf.AddLambdaNode("verifier-agent", compose.InvokableLambda(func(c context.Context, in *dagState) (*dagState, error) {
		answer := summarize(in.Planner.Intent, in.Calls)
		if e.svcCtx.Config.Orchestration.EnableVerifier {
			verified, err := e.verifierAgent(c, chatModel, in.Req, &in.Planner, in.Calls)
			if err == nil && strings.TrimSpace(verified) != "" {
				answer = strings.TrimSpace(verified)
			}
		}
		in.Answer = answer
		return in, nil
	}))
	verifierNode.AddInput("tool-agent")

	wf.End().AddInput("verifier-agent")

	compileOpts := make([]compose.GraphCompileOption, 0, 1)
	if e.svcCtx.Config.Orchestration.MaxSteps > 0 {
		compileOpts = append(compileOpts, compose.WithMaxRunSteps(e.svcCtx.Config.Orchestration.MaxSteps))
	}

	r, err := wf.Compile(ctx, compileOpts...)
	if err != nil {
		return nil, err
	}

	return r.Invoke(ctx, input)
}

// intentAgent 调用 LLM 识别用户意图。
func (e *EinoEngine) intentAgent(ctx context.Context, model model.BaseChatModel, req *types.ExecuteRequest) (string, error) {
	system := `You are intent-agent.
Return ONLY JSON with field "intent".
Allowed values: create_shortlink, update_shortlink, query_traffic, update_and_query, unknown.
If user asks to change expiration/expiry/有效期/过期时间, classify as update_shortlink.`

	resp, err := model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(system),
		schema.UserMessage(req.Query),
	})
	if err != nil {
		return "", err
	}

	var out intentDecision
	if err := decodeJSONContent(resp.Content, &out); err != nil {
		return "", err
	}

	return normalizeIntent(out.Intent), nil
}

// plannerAgent 调用 LLM 产出结构化执行计划。
func (e *EinoEngine) plannerAgent(ctx context.Context, model model.BaseChatModel, req *types.ExecuteRequest, intent string) (*plannerDecision, error) {
	system := `You are planner-agent.
Return ONLY JSON with fields:
- intent: one of create_shortlink, update_shortlink, query_traffic, update_and_query, unknown
- steps: array of short strings
- summary: one sentence`

	user := fmt.Sprintf("intent=%s\nquery=%s", intent, req.Query)

	resp, err := model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(system),
		schema.UserMessage(user),
	})
	if err != nil {
		return nil, err
	}

	var out plannerDecision
	if err := decodeJSONContent(resp.Content, &out); err != nil {
		return nil, err
	}

	if strings.TrimSpace(out.Intent) == "" {
		out.Intent = intent
	}

	return &out, nil
}

// executeToolPlan 是 tool-calling 不可用时的本地兜底执行路径。
func (e *EinoEngine) executeToolPlan(ctx context.Context, intent string, req *types.ExecuteRequest) []types.ToolCall {
	calls := make([]types.ToolCall, 0, 2)

	resolvedIntent := normalizeIntent(intent)

	switch resolvedIntent {
	case "create_shortlink":
		calls = append(calls, e.tools.CreateShortLink(ctx, req))
	case "update_shortlink":
		if !e.svcCtx.Config.Orchestration.AllowWriteActions {
			calls = append(calls, types.ToolCall{Tool: "shortener.update", Input: req.Query, Status: "blocked", Error: "write actions disabled"})
		} else {
			calls = append(calls, e.tools.UpdateShortLink(ctx, req))
		}
	case "query_traffic":
		calls = append(calls, e.tools.QueryTrafficSummary(ctx, req))
	case "update_and_query":
		if e.svcCtx.Config.Orchestration.AllowWriteActions {
			calls = append(calls, e.tools.UpdateShortLink(ctx, req))
		} else {
			calls = append(calls, types.ToolCall{Tool: "shortener.update", Input: req.Query, Status: "blocked", Error: "write actions disabled"})
		}
		calls = append(calls, e.tools.QueryTrafficSummary(ctx, req))
	default:
		calls = append(calls, types.ToolCall{Tool: "intent.fallback", Input: req.Query, Status: "unknown", Error: "unsupported intent"})
	}

	return calls
}

// toolAgent 通过 LLM tool-calling 决定调用哪些工具并执行。
func (e *EinoEngine) toolAgent(ctx context.Context, chatModel *einoOpenAI.ChatModel, state *dagState) ([]types.ToolCall, error) {
	intent := strings.TrimSpace(state.Planner.Intent)
	if intent == "" {
		intent = strings.TrimSpace(state.Intent)
	}
	intent = normalizeIntent(intent)
	if intent == "unknown" {
		if inferred, inferErr := e.intentAgent(ctx, chatModel, state.Req); inferErr == nil {
			intent = normalizeIntent(inferred)
		}
	}

	toolInfos := e.toolInfos()
	bm, err := chatModel.WithTools(toolInfos)
	if err != nil {
		return nil, err
	}

	system := `You are tool-agent.
Choose proper tools and call them by tool-calling.
Only call from: create_shortlink, update_shortlink, query_traffic.
For update_shortlink, you can update longUrl, expireAt, or both.
If user asks update+query, call both update_shortlink and query_traffic in order.`

	ctxJSON, _ := json.Marshal(state.Req.Context)
	user := fmt.Sprintf("intent=%s\nquery=%s\ncontext=%s", intent, state.Req.Query, string(ctxJSON))

	msg, err := bm.Generate(ctx, []*schema.Message{
		schema.SystemMessage(system),
		schema.UserMessage(user),
	})
	if err != nil {
		return nil, err
	}

	if len(msg.ToolCalls) == 0 {
		return e.executeToolPlan(ctx, intent, state.Req), nil
	}

	calls := make([]types.ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		calls = append(calls, e.dispatchToolCall(ctx, state.Req, tc))
	}

	if len(calls) == 0 {
		return e.executeToolPlan(ctx, intent, state.Req), nil
	}

	return calls, nil
}

// toolInfos 定义可暴露给模型的工具描述与参数。
func (e *EinoEngine) toolInfos() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{
			Name: "create_shortlink",
			Desc: "Create a short link from a long URL",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"longUrl": {
					Type:     schema.String,
					Desc:     "the long URL to shorten",
					Required: true,
				},
			}),
		},
		{
			Name: "update_shortlink",
			Desc: "Update short link target URL and/or expiration time",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"shortCode": {
					Type:     schema.String,
					Desc:     "short code to update",
					Required: true,
				},
				"longUrl": {
					Type:     schema.String,
					Desc:     "new long URL",
					Required: false,
				},
				"expireAt": {
					Type:     schema.String,
					Desc:     "expiration time, prefer RFC3339",
					Required: false,
				},
			}),
		},
		{
			Name: "query_traffic",
			Desc: "Query traffic summary of short link",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"shortCode": {
					Type:     schema.String,
					Desc:     "short code",
					Required: true,
				},
				"from": {
					Type:     schema.String,
					Desc:     "start date in YYYY-MM-DD",
					Required: true,
				},
				"to": {
					Type:     schema.String,
					Desc:     "end date in YYYY-MM-DD",
					Required: true,
				},
			}),
		},
	}
}

// dispatchToolCall 将模型返回的 ToolCall 分发到内部 HTTP 工具客户端执行。
func (e *EinoEngine) dispatchToolCall(ctx context.Context, req *types.ExecuteRequest, tc schema.ToolCall) types.ToolCall {
	args := make(map[string]any)
	_ = decodeJSONContent(tc.Function.Arguments, &args)

	reqCopy := *req
	reqCopy.Context = req.Context

	switch tc.Function.Name {
	case "create_shortlink":
		if longURL, ok := args["longUrl"].(string); ok && strings.TrimSpace(longURL) != "" {
			reqCopy.Context.LongUrl = longURL
		}
		return e.tools.CreateShortLink(ctx, &reqCopy)

	case "update_shortlink":
		if !e.svcCtx.Config.Orchestration.AllowWriteActions {
			return types.ToolCall{Tool: "shortener.update", Input: tc.Function.Arguments, Status: "blocked", Error: "write actions disabled"}
		}
		if shortCode, ok := args["shortCode"].(string); ok && strings.TrimSpace(shortCode) != "" {
			reqCopy.Context.ShortUrl = shortCode
		}
		if longURL, ok := args["longUrl"].(string); ok && strings.TrimSpace(longURL) != "" {
			reqCopy.Context.LongUrl = longURL
		}
		if expireAt, ok := args["expireAt"].(string); ok && strings.TrimSpace(expireAt) != "" {
			reqCopy.Context.ExpireAt = expireAt
		}
		return e.tools.UpdateShortLink(ctx, &reqCopy)

	case "query_traffic":
		if shortCode, ok := args["shortCode"].(string); ok && strings.TrimSpace(shortCode) != "" {
			reqCopy.Context.ShortUrl = shortCode
		}
		if from, ok := args["from"].(string); ok && strings.TrimSpace(from) != "" {
			reqCopy.Context.From = from
		}
		if to, ok := args["to"].(string); ok && strings.TrimSpace(to) != "" {
			reqCopy.Context.To = to
		}
		return e.tools.QueryTrafficSummary(ctx, &reqCopy)
	default:
		return types.ToolCall{Tool: tc.Function.Name, Input: tc.Function.Arguments, Status: "unknown", Error: "unsupported tool"}
	}
}

// verifierAgent 对计划和调用结果做最终校验与回答生成。
func (e *EinoEngine) verifierAgent(ctx context.Context, model model.BaseChatModel, req *types.ExecuteRequest, planner *plannerDecision, calls []types.ToolCall) (string, error) {
	callText, _ := json.Marshal(calls)
	planText, _ := json.Marshal(planner)

	system := `You are verifier-agent.
Given query, plan and tool call outputs, return ONLY JSON: {"answer":"..."}.
Use concise Chinese response and include failure reasons if any.`

	user := fmt.Sprintf("query=%s\nplan=%s\ncalls=%s", req.Query, string(planText), string(callText))

	resp, err := model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(system),
		schema.UserMessage(user),
	})
	if err != nil {
		return "", err
	}

	var out verifierDecision
	if err := decodeJSONContent(resp.Content, &out); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.Answer), nil
}

// nonEmptySteps 清洗模型返回的步骤列表，空时回退默认计划。
func nonEmptySteps(steps []string, fallback []string) []string {
	if len(steps) == 0 {
		return fallback
	}
	res := make([]string, 0, len(steps))
	for _, s := range steps {
		t := strings.TrimSpace(s)
		if t != "" {
			res = append(res, t)
		}
	}
	if len(res) == 0 {
		return fallback
	}
	return res
}

// decodeJSONContent 解析模型输出中的 JSON（支持带 markdown 包裹）。
func decodeJSONContent(content string, out any) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return errors.New("empty model response")
	}

	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	if err := json.Unmarshal([]byte(trimmed), out); err == nil {
		return nil
	}

	left := strings.Index(trimmed, "{")
	right := strings.LastIndex(trimmed, "}")
	if left >= 0 && right > left {
		return json.Unmarshal([]byte(trimmed[left:right+1]), out)
	}

	left = strings.Index(trimmed, "[")
	right = strings.LastIndex(trimmed, "]")
	if left >= 0 && right > left {
		return json.Unmarshal([]byte(trimmed[left:right+1]), out)
	}

	return errors.New("invalid json content")
}

// validateExecuteRequest 校验执行请求的必填字段。
func validateExecuteRequest(req *types.ExecuteRequest) error {
	if strings.TrimSpace(req.RequestId) == "" {
		return errors.New("requestId is required")
	}
	if strings.TrimSpace(req.OperatorId) == "" {
		return errors.New("operatorId is required")
	}
	if strings.TrimSpace(req.TenantId) == "" {
		return errors.New("tenantId is required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return errors.New("query is required")
	}
	return nil
}
