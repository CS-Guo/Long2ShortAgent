package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aiagent/internal/svc"
	"aiagent/internal/types"
)

type InternalClient struct {
	svcCtx *svc.ServiceContext
}

// NewInternalClient 创建内部服务调用客户端。
func NewInternalClient(svcCtx *svc.ServiceContext) *InternalClient {
	return &InternalClient{svcCtx: svcCtx}
}

// CreateShortLink 调用 shortener-core 创建短链接口。
func (c *InternalClient) CreateShortLink(ctx context.Context, req *types.ExecuteRequest) types.ToolCall {
	body := map[string]any{
		"requestId":  req.RequestId,
		"operatorId": req.OperatorId,
		"tenantId":   req.TenantId,
		"longUrl":    req.Context.LongUrl,
	}
	endpoint := c.svcCtx.Config.InternalServices.ShortenerCoreBaseURL + "/internal/shortlinks"
	return c.postJSON(ctx, "shortener.create", endpoint, body)
}

// UpdateShortLink 调用 shortener-core 修改短链接口。
func (c *InternalClient) UpdateShortLink(ctx context.Context, req *types.ExecuteRequest) types.ToolCall {
	shortCode := strings.TrimSpace(req.Context.ShortUrl)
	if shortCode == "" {
		return types.ToolCall{
			Tool:   "shortener.update",
			Input:  req.Query,
			Status: "error",
			Error:  "shortCode is required, provide context.shortUrl or reuse the same session after creating a short link",
		}
	}

	body := map[string]any{
		"requestId":      req.RequestId,
		"idempotencyKey": req.IdempotencyKey,
		"operatorId":     req.OperatorId,
		"tenantId":       req.TenantId,
	}

	if strings.TrimSpace(req.Context.LongUrl) != "" {
		body["longUrl"] = req.Context.LongUrl
	}
	if strings.TrimSpace(req.Context.ExpireAt) != "" {
		body["expireAt"] = req.Context.ExpireAt
	}

	endpoint := c.svcCtx.Config.InternalServices.ShortenerCoreBaseURL + "/internal/shortlinks/" + url.PathEscape(shortCode)
	return c.patchJSON(ctx, "shortener.update", endpoint, body)
}

// QueryTrafficSummary 调用 traffic-service 查询统计接口。
func (c *InternalClient) QueryTrafficSummary(ctx context.Context, req *types.ExecuteRequest) types.ToolCall {
	shortCode := strings.TrimSpace(req.Context.ShortUrl)
	if shortCode == "" {
		return types.ToolCall{
			Tool:   "traffic.summary",
			Input:  req.Query,
			Status: "error",
			Error:  "shortCode is required for traffic query",
		}
	}

	from := strings.TrimSpace(req.Context.From)
	to := strings.TrimSpace(req.Context.To)
	if from == "" || to == "" {
		return types.ToolCall{
			Tool:   "traffic.summary",
			Input:  req.Query,
			Status: "error",
			Error:  "from and to are required for traffic query (format: YYYY-MM-DD)",
		}
	}

	if _, err := time.Parse("2006-01-02", from); err != nil {
		return types.ToolCall{
			Tool:   "traffic.summary",
			Input:  req.Query,
			Status: "error",
			Error:  "invalid from format, expected YYYY-MM-DD",
		}
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return types.ToolCall{
			Tool:   "traffic.summary",
			Input:  req.Query,
			Status: "error",
			Error:  "invalid to format, expected YYYY-MM-DD",
		}
	}

	endpoint := fmt.Sprintf("%s/internal/traffic/shortlinks/%s/summary?from=%s&to=%s",
		c.svcCtx.Config.InternalServices.TrafficServiceBaseURL,
		url.PathEscape(shortCode),
		url.QueryEscape(from),
		url.QueryEscape(to),
	)
	return c.getJSON(ctx, "traffic.summary", endpoint)
}

// postJSON 发起 POST JSON 请求。
func (c *InternalClient) postJSON(ctx context.Context, toolName, endpoint string, payload map[string]any) types.ToolCall {
	return c.doJSON(ctx, http.MethodPost, toolName, endpoint, payload)
}

// patchJSON 发起 PATCH JSON 请求。
func (c *InternalClient) patchJSON(ctx context.Context, toolName, endpoint string, payload map[string]any) types.ToolCall {
	return c.doJSON(ctx, http.MethodPatch, toolName, endpoint, payload)
}

// getJSON 发起 GET 请求并封装为 ToolCall。
func (c *InternalClient) getJSON(ctx context.Context, toolName, endpoint string) types.ToolCall {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return types.ToolCall{Tool: toolName, Input: endpoint, Status: "error", Error: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+c.svcCtx.Config.InternalServices.ServiceToken)

	resp, err := c.svcCtx.HTTPClient.Do(req)
	if err != nil {
		return types.ToolCall{Tool: toolName, Input: endpoint, Status: "error", Error: err.Error()}
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)

	status := "ok"
	errMsg := ""
	if resp.StatusCode >= 400 {
		status = "error"
		errMsg = extractHTTPErrorMessage(buf.String())
		if errMsg == "" {
			errMsg = fmt.Sprintf("http status %d", resp.StatusCode)
		}
	}

	return types.ToolCall{
		Tool:   toolName,
		Input:  endpoint,
		Output: buf.String(),
		Status: status,
		Error:  errMsg,
	}
}

// doJSON 发起通用 JSON 请求并封装返回值。
func (c *InternalClient) doJSON(ctx context.Context, method, toolName, endpoint string, payload map[string]any) types.ToolCall {
	raw, err := json.Marshal(payload)
	if err != nil {
		return types.ToolCall{Tool: toolName, Input: endpoint, Status: "error", Error: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(raw))
	if err != nil {
		return types.ToolCall{Tool: toolName, Input: endpoint, Status: "error", Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.svcCtx.Config.InternalServices.ServiceToken)

	resp, err := c.svcCtx.HTTPClient.Do(req)
	if err != nil {
		return types.ToolCall{Tool: toolName, Input: string(raw), Status: "error", Error: err.Error()}
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)

	status := "ok"
	errMsg := ""
	if resp.StatusCode >= 400 {
		status = "error"
		errMsg = extractHTTPErrorMessage(buf.String())
		if errMsg == "" {
			errMsg = fmt.Sprintf("http status %d", resp.StatusCode)
		}
	}

	return types.ToolCall{
		Tool:   toolName,
		Input:  string(raw),
		Output: buf.String(),
		Status: status,
		Error:  errMsg,
	}
}

// extractHTTPErrorMessage 尝试从下游响应体提取业务错误信息。
func extractHTTPErrorMessage(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &payload); err == nil {
		for _, key := range []string{"msg", "message", "error", "detail"} {
			if text, ok := payload[key].(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}

	if len(trimmed) > 180 {
		return strings.TrimSpace(trimmed[:180]) + "..."
	}
	return trimmed
}
