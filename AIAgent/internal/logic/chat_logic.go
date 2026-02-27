package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"aiagent/internal/agent"
	"aiagent/internal/svc"
	"aiagent/internal/types"

	einoOpenAI "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewChatLogic 创建会话对话逻辑对象。
func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Chat 支持两种模式：
// 1) direct：仅与大模型对话；
// 2) agent：在对话中触发短链工具调用（创建/修改/查询）。
func (l *ChatLogic) Chat(req *types.ChatRequest) (*types.ChatResponse, error) {
	if strings.TrimSpace(req.RequestId) == "" {
		return nil, errors.New("requestId is required")
	}

	sessionID := l.normalizeSessionID(req)
	session, _ := l.svcCtx.GetSession(sessionID)
	mergedCtx := l.mergeContext(session, req.Context)
	userQuery := strings.TrimSpace(l.resolveUserQuery(req))
	mergedCtx = mergeExtractedContext(mergedCtx, extractContextFromQuery(userQuery))

	if l.shouldUseAgent(req, userQuery, mergedCtx) {
		return l.chatWithAgent(req, sessionID, session, mergedCtx, userQuery)
	}

	return l.chatDirect(req, sessionID, session, mergedCtx)
}

// chatWithAgent 走多智能体编排链路，可调用内部工具。
func (l *ChatLogic) chatWithAgent(req *types.ChatRequest, sessionID string, session svc.DialogueSession, mergedCtx types.ExecuteContext, userQuery string) (*types.ChatResponse, error) {
	if strings.TrimSpace(userQuery) == "" {
		return nil, errors.New("query is required when tools are enabled")
	}

	operatorID := strings.TrimSpace(req.OperatorId)
	if operatorID == "" {
		operatorID = "chat-user"
	}

	tenantID := strings.TrimSpace(req.TenantId)
	if tenantID == "" {
		tenantID = "chat-tenant"
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = req.RequestId
	}

	execReq := &types.ExecuteRequest{
		RequestId:      req.RequestId,
		OperatorId:     operatorID,
		TenantId:       tenantID,
		IdempotencyKey: idempotencyKey,
		Query:          l.enrichQueryWithContext(userQuery, mergedCtx),
		Context:        mergedCtx,
	}

	engine := agent.NewEngine(l.svcCtx)
	result, err := engine.Execute(l.ctx, execReq)
	if err != nil {
		return nil, err
	}

	// 保持与 /agent/execute 一致：保存任务快照
	l.svcCtx.SaveTask(svc.OrchestratedTask{
		TaskID:  result.TaskID,
		Status:  result.Status,
		Intent:  result.Intent,
		Plan:    result.Plan,
		Calls:   result.Calls,
		Answer:  result.Answer,
		TraceID: result.TraceID,
	})

	session = l.updateSessionAfterAgent(sessionID, session, mergedCtx, result)
	l.svcCtx.SaveSession(session)

	return &types.ChatResponse{
		RequestId: req.RequestId,
		SessionId: sessionID,
		Model:     l.svcCtx.Config.LLM.Model,
		Mode:      "agent",
		Reply:     l.rewriteAgentReply(result),
		TaskId:    result.TaskID,
		Intent:    result.Intent,
		Calls:     result.Calls,
		Memory:    l.toMemory(session),
		TraceId:   result.TraceID,
	}, nil
}

// chatDirect 仅走大模型直聊，不触发内部工具。
func (l *ChatLogic) chatDirect(req *types.ChatRequest, sessionID string, session svc.DialogueSession, mergedCtx types.ExecuteContext) (*types.ChatResponse, error) {
	messages, err := l.buildMessages(req)
	if err != nil {
		return nil, err
	}

	timeoutMs := l.svcCtx.Config.LLM.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 8000
	}
	chatCtx, cancel := context.WithTimeout(l.ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	chatModel, err := l.newChatModel(chatCtx)
	if err != nil {
		return nil, err
	}

	resp, err := chatModel.Generate(chatCtx, messages)
	if err != nil {
		return nil, err
	}

	reply := strings.TrimSpace(resp.Content)
	if reply == "" {
		return nil, errors.New("empty model response")
	}

	session = l.updateSessionFromContext(sessionID, session, mergedCtx)
	l.svcCtx.SaveSession(session)

	return &types.ChatResponse{
		RequestId: req.RequestId,
		SessionId: sessionID,
		Model:     l.svcCtx.Config.LLM.Model,
		Mode:      "direct",
		Reply:     reply,
		Memory:    l.toMemory(session),
	}, nil
}

// shouldUseAgent 判断当前对话是否应触发工具编排。
func (l *ChatLogic) shouldUseAgent(req *types.ChatRequest, query string, ctx types.ExecuteContext) bool {
	if req.EnableTools {
		return true
	}

	if strings.TrimSpace(req.OperatorId) != "" || strings.TrimSpace(req.TenantId) != "" || strings.TrimSpace(req.IdempotencyKey) != "" {
		return true
	}

	if strings.TrimSpace(ctx.ShortUrl) != "" || strings.TrimSpace(ctx.LongUrl) != "" || strings.TrimSpace(ctx.ExpireAt) != "" || strings.TrimSpace(ctx.From) != "" || strings.TrimSpace(ctx.To) != "" {
		return true
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}

	keywords := []string{
		"短链", "短链接", "shortlink", "short link", "创建", "生成", "修改", "更新",
		"流量", "统计", "pv", "uv", "点击", "访问量", "重定向",
		"过期", "有效期", "expire", "expiration",
	}

	for _, keyword := range keywords {
		if strings.Contains(q, keyword) {
			return true
		}
	}

	return false
}

// resolveUserQuery 获取用户本轮输入：优先 query，其次 messages 最后一条 user 消息。
func (l *ChatLogic) resolveUserQuery(req *types.ChatRequest) string {
	if query := strings.TrimSpace(req.Query); query != "" {
		return query
	}

	for i := len(req.Messages) - 1; i >= 0; i-- {
		role := strings.ToLower(strings.TrimSpace(req.Messages[i].Role))
		content := strings.TrimSpace(req.Messages[i].Content)
		if content == "" {
			continue
		}
		if role == "user" || role == "" {
			return content
		}
	}

	return ""
}

// normalizeSessionID 标准化会话 ID；未传时默认使用 requestId 派生。
func (l *ChatLogic) normalizeSessionID(req *types.ChatRequest) string {
	sessionID := strings.TrimSpace(req.SessionId)
	if sessionID == "" {
		return "session-" + strings.TrimSpace(req.RequestId)
	}
	return sessionID
}

// mergeContext 将请求上下文与会话记忆合并（请求优先）。
func (l *ChatLogic) mergeContext(session svc.DialogueSession, reqCtx types.ExecuteContext) types.ExecuteContext {
	ctx := types.ExecuteContext{
		ShortUrl: strings.TrimSpace(reqCtx.ShortUrl),
		LongUrl:  strings.TrimSpace(reqCtx.LongUrl),
		ExpireAt: strings.TrimSpace(reqCtx.ExpireAt),
		From:     strings.TrimSpace(reqCtx.From),
		To:       strings.TrimSpace(reqCtx.To),
	}

	if ctx.ShortUrl == "" {
		ctx.ShortUrl = strings.TrimSpace(session.LastShortCode)
	}
	if ctx.LongUrl == "" {
		ctx.LongUrl = strings.TrimSpace(session.LastLongURL)
	}
	if ctx.ExpireAt == "" {
		ctx.ExpireAt = strings.TrimSpace(session.LastExpireAt)
	}
	if ctx.From == "" {
		ctx.From = strings.TrimSpace(session.LastFrom)
	}
	if ctx.To == "" {
		ctx.To = strings.TrimSpace(session.LastTo)
	}

	return ctx
}

// enrichQueryWithContext 给 query 附带会话上下文提示，便于意图识别与参数补全。
func (l *ChatLogic) enrichQueryWithContext(query string, ctx types.ExecuteContext) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}

	hints := make([]string, 0, 4)
	if strings.TrimSpace(ctx.ShortUrl) != "" {
		hints = append(hints, "短码="+ctx.ShortUrl)
	}
	if strings.TrimSpace(ctx.LongUrl) != "" {
		hints = append(hints, "长链="+ctx.LongUrl)
	}
	if strings.TrimSpace(ctx.ExpireAt) != "" {
		hints = append(hints, "expireAt="+ctx.ExpireAt)
	}
	if strings.TrimSpace(ctx.From) != "" {
		hints = append(hints, "from="+ctx.From)
	}
	if strings.TrimSpace(ctx.To) != "" {
		hints = append(hints, "to="+ctx.To)
	}

	if len(hints) == 0 {
		return query
	}

	return query + "\n\n已知上下文: " + strings.Join(hints, "; ")
}

// updateSessionAfterAgent 更新会话记忆（上下文 + 工具调用结果）。
func (l *ChatLogic) updateSessionAfterAgent(sessionID string, session svc.DialogueSession, mergedCtx types.ExecuteContext, result *agent.ExecuteResult) svc.DialogueSession {
	session = l.updateSessionFromContext(sessionID, session, mergedCtx)
	session.LastTaskID = result.TaskID
	session.LastIntent = result.Intent

	for _, call := range result.Calls {
		if call.Status != "ok" {
			continue
		}

		payload := map[string]any{}
		if err := json.Unmarshal([]byte(call.Output), &payload); err != nil {
			continue
		}

		switch call.Tool {
		case "shortener.create":
			if shortCode := asMapString(payload, "shortCode"); shortCode != "" {
				session.LastShortCode = shortCode
			}
			if shortURL := asMapString(payload, "shortUrl"); shortURL != "" {
				session.LastShortURL = shortURL
			}
		case "shortener.update":
			if shortCode := asMapString(payload, "shortCode"); shortCode != "" {
				session.LastShortCode = shortCode
			}
			if longURL := asMapString(payload, "longUrl"); longURL != "" {
				session.LastLongURL = longURL
			}
			if expireAt := asMapString(payload, "expireAt"); expireAt != "" {
				session.LastExpireAt = expireAt
			}
		}
	}

	session.UpdatedAtUnix = time.Now().Unix()
	return session
}

// updateSessionFromContext 仅依据上下文更新记忆。
func (l *ChatLogic) updateSessionFromContext(sessionID string, session svc.DialogueSession, ctx types.ExecuteContext) svc.DialogueSession {
	if session.SessionID == "" {
		session.SessionID = sessionID
	}

	if strings.TrimSpace(ctx.ShortUrl) != "" {
		session.LastShortCode = strings.TrimSpace(ctx.ShortUrl)
	}
	if strings.TrimSpace(ctx.LongUrl) != "" {
		session.LastLongURL = strings.TrimSpace(ctx.LongUrl)
	}
	if strings.TrimSpace(ctx.ExpireAt) != "" {
		session.LastExpireAt = strings.TrimSpace(ctx.ExpireAt)
	}
	if strings.TrimSpace(ctx.From) != "" {
		session.LastFrom = strings.TrimSpace(ctx.From)
	}
	if strings.TrimSpace(ctx.To) != "" {
		session.LastTo = strings.TrimSpace(ctx.To)
	}

	session.UpdatedAtUnix = time.Now().Unix()
	return session
}

// toMemory 将会话记忆映射为对外可见的上下文片段。
func (l *ChatLogic) toMemory(session svc.DialogueSession) types.ExecuteContext {
	return types.ExecuteContext{
		ShortUrl: session.LastShortCode,
		LongUrl:  session.LastLongURL,
		ExpireAt: session.LastExpireAt,
		From:     session.LastFrom,
		To:       session.LastTo,
	}
}

// asMapString 从 map 中安全读取字符串字段。
func asMapString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

// rewriteAgentReply 将工具调用结果重组为更面向用户的自然语言回复。
func (l *ChatLogic) rewriteAgentReply(result *agent.ExecuteResult) string {
	if result == nil {
		return "本次对话未产生可用结果。"
	}

	if len(result.Calls) == 0 {
		if strings.TrimSpace(result.Answer) != "" {
			return result.Answer
		}
		return "未触发任何工具调用，请补充更具体的信息。"
	}

	for _, call := range result.Calls {
		if call.Status == "ok" {
			continue
		}

		errLower := strings.ToLower(strings.TrimSpace(call.Error))

		if strings.Contains(errLower, "shortcode is required") {
			return "我识别到你要修改短链，但缺少短链码。请在对话里补充 shortCode（例如 a），或沿用包含短码的同一 session。"
		}
		if strings.Contains(errLower, "at least one field to update") {
			return "我识别到你要更新短链，但缺少可修改字段。请补充 longUrl 或 expireAt。"
		}
		if strings.Contains(errLower, "invalid expireat format") {
			return "过期时间格式不正确。请使用 RFC3339 或 `YYYY-MM-DD HH:MM:SS`（例如 `2026-12-31 23:59:59`）。"
		}
		if strings.Contains(errLower, "from and to are required") {
			return "查询流量需要时间范围，请补充 from 和 to（格式 `YYYY-MM-DD`）。"
		}
		if strings.Contains(errLower, "invalid from format") || strings.Contains(errLower, "invalid to format") {
			return "时间范围格式不正确，请使用 `YYYY-MM-DD`，例如 from=`2026-02-01`，to=`2026-02-11`。"
		}
		if strings.Contains(errLower, "write actions disabled") {
			return "当前环境已禁用写操作（创建/修改）。如需更新短链，请先在配置中开启写操作权限。"
		}
		if strings.Contains(errLower, "in sql, but only") && strings.Contains(errLower, "arguments provided") {
			return "短链服务内部执行异常（SQL 参数数量不匹配）。请重启最新版本的 shortener 服务后重试。"
		}
		if strings.Contains(errLower, "field \"longurl\" is not set") {
			return "更新失败：下游服务仍要求 longUrl。请确认 shortener 已升级到支持仅修改 expireAt 的版本并重启。"
		}

		errMsg := strings.TrimSpace(call.Error)
		if errMsg == "" {
			errMsg = strings.TrimSpace(call.Output)
		}
		if errMsg == "" {
			errMsg = "工具调用失败，但没有返回明确错误信息。"
		}
		return "已识别到你的意图，但执行失败：" + errMsg
	}

	for _, call := range result.Calls {
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(call.Output), &payload)
		if nested, ok := payload["data"].(map[string]any); ok && len(nested) > 0 {
			payload = nested
		}

		switch call.Tool {
		case "shortener.update":
			shortCode := asMapString(payload, "shortCode")
			longURL := asMapString(payload, "longUrl")
			expireAt := asMapString(payload, "expireAt")

			parts := make([]string, 0, 2)
			if longURL != "" {
				parts = append(parts, "目标链接已更新")
			}
			if expireAt != "" {
				parts = append(parts, "过期时间已更新为 "+expireAt)
			}
			if len(parts) > 0 {
				if shortCode != "" {
					return "更新成功：短链码 " + shortCode + "，" + strings.Join(parts, "，")
				}
				return "更新成功：" + strings.Join(parts, "，")
			}
			if shortCode != "" {
				return "已定位短链码 " + shortCode + "，但未检测到可更新字段。请明确提供 longUrl 或 expireAt。"
			}
			return "短链更新请求已执行，但未返回可识别的更新结果。"
		case "shortener.create":
			shortURL := asMapString(payload, "shortUrl")
			shortCode := asMapString(payload, "shortCode")
			if shortURL != "" {
				if shortCode != "" {
					return "创建成功：" + shortURL + "（短码：" + shortCode + "）"
				}
				return "创建成功：" + shortURL
			}
			return "短链创建成功。"
		case "traffic.summary":
			pv, pvOK := payload["pv"]
			uv, uvOK := payload["uv"]
			if pvOK || uvOK {
				return "流量查询完成：PV=" + stringifyAny(pv) + "，UV=" + stringifyAny(uv)
			}
		}
	}

	if strings.TrimSpace(result.Answer) != "" {
		return result.Answer
	}
	return "执行完成。"
}

// stringifyAny 将动态值转成可展示字符串。
func stringifyAny(value any) string {
	if value == nil {
		return "0"
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

// mergeExtractedContext 将 query 里提取出的参数合并到上下文（仅补空，不覆盖显式传参）。
func mergeExtractedContext(base, extracted types.ExecuteContext) types.ExecuteContext {
	if strings.TrimSpace(base.ShortUrl) == "" {
		base.ShortUrl = strings.TrimSpace(extracted.ShortUrl)
	}
	if strings.TrimSpace(base.LongUrl) == "" {
		base.LongUrl = strings.TrimSpace(extracted.LongUrl)
	}
	if strings.TrimSpace(base.ExpireAt) == "" {
		base.ExpireAt = strings.TrimSpace(extracted.ExpireAt)
	}
	if strings.TrimSpace(base.From) == "" {
		base.From = strings.TrimSpace(extracted.From)
	}
	if strings.TrimSpace(base.To) == "" {
		base.To = strings.TrimSpace(extracted.To)
	}
	return base
}

// extractContextFromQuery 尝试从自然语言 query 中提取常见参数。
func extractContextFromQuery(query string) types.ExecuteContext {
	q := strings.TrimSpace(query)
	if q == "" {
		return types.ExecuteContext{}
	}

	ctx := types.ExecuteContext{}

	shortCodePattern := regexp.MustCompile(`(?:短链|短码)\s*([a-zA-Z0-9_-]{1,32})`)
	if matched := shortCodePattern.FindStringSubmatch(q); len(matched) > 1 {
		ctx.ShortUrl = matched[1]
	}
	if ctx.ShortUrl == "" {
		shortCodeKVPattern := regexp.MustCompile(`(?:shortCode|短链码|短链接码)\s*[:：=]?\s*([a-zA-Z0-9_-]{1,32})`)
		if matched := shortCodeKVPattern.FindStringSubmatch(q); len(matched) > 1 {
			ctx.ShortUrl = matched[1]
		}
	}
	if ctx.ShortUrl == "" {
		domainShortCodePattern := regexp.MustCompile(`(?:https?://)?[a-zA-Z0-9.-]+/([a-zA-Z0-9_-]{1,32})`)
		if matched := domainShortCodePattern.FindStringSubmatch(q); len(matched) > 1 {
			ctx.ShortUrl = matched[1]
		}
	}

	urlPattern := regexp.MustCompile(`https?://[^\s，。]+`)
	if matched := urlPattern.FindString(q); matched != "" {
		ctx.LongUrl = matched
	}

	timePattern := regexp.MustCompile(`\d{4}-\d{2}-\d{2}(?:[ T]\d{2}:\d{2}:\d{2})?`)
	if matched := timePattern.FindString(q); matched != "" {
		if strings.Contains(strings.ToLower(q), "过期") || strings.Contains(strings.ToLower(q), "expire") {
			ctx.ExpireAt = matched
		}
	}
	if strings.TrimSpace(ctx.ExpireAt) == "" {
		ctx.ExpireAt = parseRelativeExpireAt(q)
	}

	dateRangePattern := regexp.MustCompile(`从\s*(\d{4}-\d{2}-\d{2})\s*到\s*(\d{4}-\d{2}-\d{2})`)
	if matched := dateRangePattern.FindStringSubmatch(q); len(matched) > 2 {
		ctx.From = matched[1]
		ctx.To = matched[2]
		return ctx
	}

	fromPattern := regexp.MustCompile(`(?:from|开始|起始)\s*[:：=]?\s*(\d{4}-\d{2}-\d{2})`)
	if matched := fromPattern.FindStringSubmatch(strings.ToLower(q)); len(matched) > 1 {
		ctx.From = matched[1]
	}

	toPattern := regexp.MustCompile(`(?:to|结束|截止)\s*[:：=]?\s*(\d{4}-\d{2}-\d{2})`)
	if matched := toPattern.FindStringSubmatch(strings.ToLower(q)); len(matched) > 1 {
		ctx.To = matched[1]
	}

	if strings.TrimSpace(ctx.From) == "" || strings.TrimSpace(ctx.To) == "" {
		relativeFrom, relativeTo := parseRelativeRange(q)
		if strings.TrimSpace(ctx.From) == "" {
			ctx.From = relativeFrom
		}
		if strings.TrimSpace(ctx.To) == "" {
			ctx.To = relativeTo
		}
	}

	return ctx
}

// parseRelativeExpireAt 解析口语化过期时间，如“七天后/3天后/明天”。
func parseRelativeExpireAt(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}

	if !strings.Contains(strings.ToLower(q), "过期") && !strings.Contains(strings.ToLower(q), "有效期") && !strings.Contains(strings.ToLower(q), "expire") {
		return ""
	}

	now := time.Now()

	daysLaterPattern := regexp.MustCompile(`([0-9一二三四五六七八九十两百]+)\s*天后`)
	if matched := daysLaterPattern.FindStringSubmatch(q); len(matched) > 1 {
		if days, ok := parseChineseOrArabicInt(matched[1]); ok && days > 0 {
			target := now.AddDate(0, 0, days)
			target = time.Date(target.Year(), target.Month(), target.Day(), 23, 59, 59, 0, target.Location())
			return target.Format("2006-01-02 15:04:05")
		}
	}

	if strings.Contains(q, "明天") {
		target := now.AddDate(0, 0, 1)
		target = time.Date(target.Year(), target.Month(), target.Day(), 23, 59, 59, 0, target.Location())
		return target.Format("2006-01-02 15:04:05")
	}

	if strings.Contains(q, "后天") {
		target := now.AddDate(0, 0, 2)
		target = time.Date(target.Year(), target.Month(), target.Day(), 23, 59, 59, 0, target.Location())
		return target.Format("2006-01-02 15:04:05")
	}

	return ""
}

// parseRelativeRange 解析口语化时间区间，如“近七天/最近7天/过去3天”。
func parseRelativeRange(query string) (from string, to string) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", ""
	}

	pattern := regexp.MustCompile(`(?:近|最近|近期|过去)\s*([0-9一二三四五六七八九十两百]+)\s*天`)
	matched := pattern.FindStringSubmatch(q)
	if len(matched) <= 1 {
		return "", ""
	}

	days, ok := parseChineseOrArabicInt(matched[1])
	if !ok || days <= 0 {
		return "", ""
	}

	now := time.Now()
	toDate := now.Format("2006-01-02")
	fromDate := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	return fromDate, toDate
}

// parseChineseOrArabicInt 解析阿拉伯数字与常见中文数字。
func parseChineseOrArabicInt(raw string) (int, bool) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, false
	}

	if n, err := strconv.Atoi(t); err == nil {
		return n, true
	}

	if n, ok := parseChineseNumber(t); ok {
		return n, true
	}

	return 0, false
}

// parseChineseNumber 解析有限集合中文数字（覆盖“七/十/十二/二十/两百”这类常见表达）。
func parseChineseNumber(text string) (int, bool) {
	t := strings.TrimSpace(text)
	if t == "" {
		return 0, false
	}

	mapDigit := map[rune]int{
		'零': 0,
		'一': 1,
		'二': 2,
		'两': 2,
		'三': 3,
		'四': 4,
		'五': 5,
		'六': 6,
		'七': 7,
		'八': 8,
		'九': 9,
	}

	// 处理 “十” 系列：十、十一、二十、二十一
	if strings.ContainsRune(t, '十') {
		runes := []rune(t)
		tenIdx := -1
		for i, r := range runes {
			if r == '十' {
				tenIdx = i
				break
			}
		}
		if tenIdx < 0 {
			return 0, false
		}

		tens := 1
		if tenIdx > 0 {
			v, ok := mapDigit[runes[tenIdx-1]]
			if !ok {
				return 0, false
			}
			tens = v
		}

		ones := 0
		if tenIdx < len(runes)-1 {
			v, ok := mapDigit[runes[tenIdx+1]]
			if !ok {
				return 0, false
			}
			ones = v
		}

		return tens*10 + ones, true
	}

	// 处理 “百” 系列：一百、两百
	if strings.ContainsRune(t, '百') {
		runes := []rune(t)
		if len(runes) >= 2 && runes[1] == '百' {
			v, ok := mapDigit[runes[0]]
			if !ok {
				return 0, false
			}
			return v * 100, true
		}
		return 0, false
	}

	if len([]rune(t)) == 1 {
		if v, ok := mapDigit[[]rune(t)[0]]; ok {
			return v, true
		}
	}

	return 0, false
}

// newChatModel 创建 Eino OpenAI ChatModel 客户端。
func (l *ChatLogic) newChatModel(ctx context.Context) (*einoOpenAI.ChatModel, error) {
	cfg := l.svcCtx.Config.LLM
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("llm base url is required")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("llm api key is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("llm model is required")
	}

	return einoOpenAI.NewChatModel(ctx, &einoOpenAI.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	})
}

// buildMessages 组装直聊模式消息：支持 messages（多轮）和 query（单轮快捷）。
func (l *ChatLogic) buildMessages(req *types.ChatRequest) ([]*schema.Message, error) {
	messages := make([]*schema.Message, 0, len(req.Messages)+2)

	if system := strings.TrimSpace(req.SystemPrompt); system != "" {
		messages = append(messages, schema.SystemMessage(system))
	}

	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}

		switch role {
		case "system":
			messages = append(messages, schema.SystemMessage(content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(content, nil))
		case "user", "":
			messages = append(messages, schema.UserMessage(content))
		default:
			return nil, errors.New("unsupported role, only system/user/assistant are allowed")
		}
	}

	if query := strings.TrimSpace(req.Query); query != "" {
		messages = append(messages, schema.UserMessage(query))
	}

	if len(messages) == 0 {
		return nil, errors.New("messages or query is required")
	}

	return messages, nil
}
