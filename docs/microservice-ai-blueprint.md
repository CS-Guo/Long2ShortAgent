# Long2Short 多微服务 + 多智能体蓝图（MVP）

本文档给出你当前项目（`Shortener`）向“AI 编排微服务 + 流量统计微服务”演进的最小可落地方案。

## 1. 服务边界

### 1.1 shortener-core（现有服务）
- 职责：短链创建、修改、删除、跳转。
- 数据：`short_url_map` 主数据唯一来源。
- 约束：所有写操作（create/update/delete）都必须经过本服务。

### 1.2 traffic-service（新增）
- 职责：消费点击事件，聚合 PV/UV/趋势/来源数据。
- 数据：`short_url_click_event`（明细，可选），`short_url_daily_stat`（聚合）。
- 对外：查询统计接口（给 AI 编排服务和控制台使用）。

### 1.3 ai-orchestrator（新增）
- 职责：自然语言理解 + 多智能体协同编排。
- 约束：不直接写数据库，只调用内部工具 API。
- 对外：统一入口 `POST /agent/execute`。

## 2. 多智能体协同流程

### 2.1 Agent 角色
- `intent-agent`：识别用户意图（创建短链 / 修改短链 / 查询流量）。
- `planner-agent`：构建执行计划（步骤、依赖、回滚点）。
- `tool-agent`：按计划调用内部工具（shortener-core / traffic-service）。
- `verifier-agent`：检查执行结果、权限、幂等，给出最终响应。

### 2.2 执行链路（示例："把 A 短链改成 B 并看近7天流量"）
1. 用户请求进入 `ai-orchestrator`。
2. `intent-agent` 输出结构化意图：`update_shortlink + query_traffic`。
3. `planner-agent` 生成计划：先更新 -> 再查询 -> 汇总。
4. `tool-agent` 调用 `shortener-core PATCH /internal/shortlinks/{surl}`。
5. `tool-agent` 调用 `traffic-service GET /internal/traffic/shortlinks/{surl}/summary`。
6. `verifier-agent` 校验结果后返回最终答复。

## 3. 统一数据契约

### 3.1 命令幂等与审计
- 每个 AI 请求必须携带：`requestId`、`operatorId`、`tenantId`。
- 修改类操作必须携带：`idempotencyKey`。
- 所有 AI 写操作必须写 `audit_log`（可落 DB 或日志平台）。

### 3.2 追踪字段（推荐）
- `traceId`：跨服务链路追踪。
- `spanId`：服务内追踪。
- `taskId`：AI 任务 ID（一次自然语言请求）。

## 4. 安全与权限

- `ai-orchestrator` 调内部 API 必须使用内网鉴权（例如 HMAC/JWT service token）。
- 所有“修改短链”操作必须做权限校验（资源所有者 / 租户隔离）。
- AI 不允许直连 MySQL，只能通过工具 API。

## 5. MVP 迭代顺序

1. **Phase A**：shortener-core 增加“修改/删除短链”内部接口。
2. **Phase B**：shortener-core 在跳转处写入 Redis Stream 点击事件。
3. **Phase C**：traffic-service 消费事件并提供查询接口。
4. **Phase D**：ai-orchestrator 接入 4-Agent 流程和工具调用。
5. **Phase E**：补全观测、限流、熔断、审计。

## 6. 目录建议

```text
Long2ShortAgent/
├── Shortener/                 # shortener-core
├── TrafficService/            # 新增：流量统计服务
├── AIAgent/                   # 新增：AI 编排服务
├── contracts/                 # 事件/内部 API 契约
└── docs/                      # 架构文档
```

