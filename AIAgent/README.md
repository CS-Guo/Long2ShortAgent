# AIAgent（AI 编排微服务）

## 目标

`AIAgent` 负责把自然语言请求转为可执行的内部工具调用，支持：
- 创建短链
- 修改短链
- 查询短链流量

## 多智能体角色

- `intent-agent`：意图识别（create/update/query），统一由大模型判断，不再使用关键词规则匹配。
- `planner-agent`：生成执行计划与调用顺序。
- `tool-agent`：调用内部工具 API（shortener-core / traffic-service）。
- `verifier-agent`：结果校验、权限确认、响应拼装。

## 对外接口

- `POST /agent/execute`：多智能体编排入口（可调用内部工具）。
- `POST /agent/chat`：会话对话入口（可直聊，也可在对话中触发短链工具）。
- `GET /agent/tasks/:taskId`：查询任务执行详情。

## 核心原则

- AI 服务不直连业务数据库。
- 写操作仅通过 `shortener-core` 内部 API。
- 每次执行都要携带 `requestId` 和 `idempotencyKey`。

## 引擎说明

- 当前支持通过 `Orchestration.Engine` 选择编排引擎。
- `rule`：默认，可运行的规则编排引擎（用于开发联调）。
- `eino`：已接入 Eino 多智能体链路（intent/planner/verifier + tool-agent）。
- 当 `eino` 执行失败时，可通过 `FallbackToRule` 降级到 `rule`。
- `eino` 当前实现为 compose Workflow DAG：`intent-agent -> planner-agent -> tool-agent(tool-calling) -> verifier-agent`。


## 直聊示例

```bash
curl -X POST http://127.0.0.1:8890/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "requestId":"chat-001",
    "systemPrompt":"你是一个简洁的中文助手",
    "messages":[
      {"role":"user","content":"你好，帮我介绍一下这个项目"}
    ]
  }'
```

也支持单轮快捷字段：`query`。


## 会话式短链示例

### 1) 创建短链（开启工具模式）

```bash
curl -X POST http://127.0.0.1:8890/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "requestId":"chat-create-001",
    "sessionId":"demo-session-1",
    "enableTools":true,
    "operatorId":"u1001",
    "tenantId":"t1001",
    "query":"帮我创建一个短链",
    "context":{"longUrl":"https://example.com/products/1001"}
  }'
```

### 2) 基于同一会话继续修改（可不再重复 shortCode）

```bash
curl -X POST http://127.0.0.1:8890/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "requestId":"chat-update-001",
    "sessionId":"demo-session-1",
    "enableTools":true,
    "operatorId":"u1001",
    "tenantId":"t1001",
    "query":"把刚才那个短链改成跳转到 https://example.com/new-page"
  }'
```

### 2.1) 通过对话修改过期时间

```bash
curl -X POST http://127.0.0.1:8890/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "requestId":"chat-expire-001",
    "sessionId":"demo-session-1",
    "enableTools":true,
    "operatorId":"u1001",
    "tenantId":"t1001",
    "query":"把这个短链的过期时间改到 2026-12-31 23:59:59",
    "context":{"expireAt":"2026-12-31 23:59:59"}
  }'
```

### 3) 在同一会话查询流量

```bash
curl -X POST http://127.0.0.1:8890/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "requestId":"chat-traffic-001",
    "sessionId":"demo-session-1",
    "enableTools":true,
    "operatorId":"u1001",
    "tenantId":"t1001",
    "query":"查询这个短链最近7天流量",
    "context":{"from":"2026-02-05","to":"2026-02-11"}
  }'
```

返回体中的 `mode=agent` 表示已走工具编排，`memory` 字段是当前会话记忆（短码/时间范围等）。


## 流式对话示例

```bash
curl -N -X POST http://127.0.0.1:8890/agent/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "requestId":"chat-stream-001",
    "sessionId":"demo-session-1",
    "enableTools":true,
    "operatorId":"u1001",
    "tenantId":"t1001",
    "query":"继续查询这个短链最近7天的流量"
  }'
```

SSE 事件说明：
- `meta`：本轮元信息（mode/taskId/intent/memory）。
- `delta`：回复增量分片。
- `done`：完整结果（与 `/agent/chat` 返回体一致）。

## 命令行直连大模型（CLI）

除了 HTTP 接口，你也可以直接在命令行与模型对话。

### 1) 仅大模型直聊（不调用工具）

```bash
go run agent.go -f etc/ai-agent.yaml -cli
```

### 2) 开启工具编排（可创建/修改短链、查流量）

```bash
go run agent.go -f etc/ai-agent.yaml -cli -cli-tools \
  -cli-session demo-cli-1 \
  -cli-operator u1001 \
  -cli-tenant t1001
```

### 3) CLI 内置命令

- `/help`：查看帮助
- `/tools on|off`：切换工具编排
- `/memory`：查看当前 session 记忆（短码、时间范围等）
- `/new`：新建 session 并清空本地历史
- `/reset`：清空本地对话历史
- `/quit`：退出

说明：
- 在 `-cli-tools` 模式下，系统会优先用 `query` + `session memory` 自动补参数。
- 若参数缺失/格式不正确，会返回明确中文提示。

## 超短命令

如果你不想记一长串 `go run` 参数，直接在 `AIAgent` 目录使用：

```bash
./chat
```

开启工具编排模式：

```bash
./chat --tools
```

指定会话：

```bash
./chat --tools --session demo-cli-1
```
