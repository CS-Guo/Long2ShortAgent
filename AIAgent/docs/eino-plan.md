# AIAgent Eino 实施计划

本文档描述如何将当前 `rule` 引擎替换为 `eino` 引擎，同时保持外部 API 与内部工具契约不变。

## 1. 当前状态

- 已有引擎抽象：`internal/agent/engine.go`
- 已有默认实现：`RuleEngine`
- 已有 Eino 实现：`internal/agent/eino_engine.go`
- 通过配置切换：`Orchestration.Engine`
- 支持失败降级：`Orchestration.FallbackToRule`

## 2. Eino 目标架构

使用 Eino 实现多智能体 DAG：

1. `intent-agent` 节点：自然语言 -> 结构化意图。
2. `planner-agent` 节点：意图 -> 步骤计划（工具与参数）。
3. `tool-agent` 节点：执行工具调用（短链创建/修改、流量查询）。
4. `verifier-agent` 节点：结果验证与最终答复生成。

## 3. 建议目录（后续新增）

```text
internal/agent/eino/
├── engine.go          # 实现 Engine 接口
├── graph.go           # Eino DAG 构建
├── nodes_intent.go    # intent 节点
├── nodes_planner.go   # planner 节点
├── nodes_tool.go      # tool 节点
└── nodes_verifier.go  # verifier 节点
```

## 4. 迁移原则

- `POST /agent/execute` 出参不变。
- `ToolCall` 结构不变。
- Internal API 契约不变。
- 若 Eino 失败，允许降级到 `rule`（可配置）。

## 5. 下一步优化

1. 将 `intent/planner/verifier` 的 JSON 解析切换为结构化输出模式（JSON Schema）。
2. 将 tool-agent 从手动 switch 升级为 Eino tool calling。
3. 增加多轮上下文（task memory）和审计字段。
4. 在测试环境以 `Engine=eino` 灰度，失败走 `FallbackToRule=true`。
5. 稳定后关闭降级并固化 Eino 为默认。
