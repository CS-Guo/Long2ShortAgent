package types

// ExecuteContext 是执行请求中的补充上下文参数。
type ExecuteContext struct {
	ShortUrl string `json:"shortUrl,optional"`
	LongUrl  string `json:"longUrl,optional"`
	ExpireAt string `json:"expireAt,optional"`
	From     string `json:"from,optional"`
	To       string `json:"to,optional"`
}

// ExecuteRequest 是统一执行入口请求体。
type ExecuteRequest struct {
	RequestId      string         `json:"requestId"`
	OperatorId     string         `json:"operatorId"`
	TenantId       string         `json:"tenantId"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Query          string         `json:"query"`
	Context        ExecuteContext `json:"context,optional"`
}

// ToolCall 记录一次工具调用信息。
type ToolCall struct {
	Tool   string `json:"tool"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Status string `json:"status"`
	Error  string `json:"error,optional"`
}

// ExecuteResponse 是统一执行入口返回体。
type ExecuteResponse struct {
	TaskId  string     `json:"taskId"`
	Intent  string     `json:"intent"`
	Answer  string     `json:"answer"`
	Calls   []ToolCall `json:"calls"`
	TraceId string     `json:"traceId"`
}

// TaskDetailRequest 是任务详情查询请求。
type TaskDetailRequest struct {
	TaskId string `path:"taskId"`
}

// TaskDetailResponse 是任务详情查询返回。
type TaskDetailResponse struct {
	TaskId  string     `json:"taskId"`
	Status  string     `json:"status"`
	Intent  string     `json:"intent"`
	Plan    []string   `json:"plan"`
	Calls   []ToolCall `json:"calls"`
	Answer  string     `json:"answer"`
	TraceId string     `json:"traceId"`
}

// ChatMessage 表示一条对话消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 是会话对话请求体：支持纯聊天，也支持触发短链工具调用。
type ChatRequest struct {
	RequestId      string         `json:"requestId"`
	SessionId      string         `json:"sessionId,optional"`
	OperatorId     string         `json:"operatorId,optional"`
	TenantId       string         `json:"tenantId,optional"`
	IdempotencyKey string         `json:"idempotencyKey,optional"`
	EnableTools    bool           `json:"enableTools,optional"`
	SystemPrompt   string         `json:"systemPrompt,optional"`
	Query          string         `json:"query,optional"`
	Messages       []ChatMessage  `json:"messages,optional"`
	Context        ExecuteContext `json:"context,optional"`
}

// ChatResponse 是会话对话返回体。
type ChatResponse struct {
	RequestId string         `json:"requestId"`
	SessionId string         `json:"sessionId"`
	Model     string         `json:"model"`
	Mode      string         `json:"mode"`
	Reply     string         `json:"reply"`
	TaskId    string         `json:"taskId,optional"`
	Intent    string         `json:"intent,optional"`
	Calls     []ToolCall     `json:"calls,optional"`
	Memory    ExecuteContext `json:"memory,optional"`
	TraceId   string         `json:"traceId,optional"`
}
