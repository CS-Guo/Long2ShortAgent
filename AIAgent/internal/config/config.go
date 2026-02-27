package config

import "github.com/zeromicro/go-zero/rest"

// Config 定义 ai-agent 的总配置结构。
type Config struct {
	rest.RestConf

	// LLM 配置大模型连接参数。
	LLM struct {
		BaseURL   string // 大模型网关地址
		APIKey    string // 大模型访问密钥
		Model     string // 模型名称
		TimeoutMs int    // 调用超时（毫秒）
	}

	// InternalServices 配置内部微服务调用地址。
	InternalServices struct {
		ShortenerCoreBaseURL  string // 短链核心服务地址
		TrafficServiceBaseURL string // 流量统计服务地址
		ServiceToken          string // 内部服务鉴权 Token
		TimeoutMs             int    // 内部调用超时（毫秒）
	}

	// Orchestration 配置编排策略。
	Orchestration struct {
		Engine            string // 使用的引擎：rule / eino
		MaxSteps          int    // DAG 最大执行步数
		EnableVerifier    bool   // 是否启用 verifier-agent
		AllowWriteActions bool   // 是否允许写操作（创建/修改）
		FallbackToRule    bool   // Eino 失败是否降级到 rule
	}
}
