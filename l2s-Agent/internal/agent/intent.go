package agent

import "l2s-Agent/pkg"

// Intent 意图识别模块
type Intent string

const (
	IntentCreated Intent = "create_short"
	IntentQuery   Intent = "query_short"
	IntentChat    Intent = "chat"
)

// DetectIntent 意图识别
func DetectIntent(intent string) Intent {
	switch {
	case pkg.Contains(intent, []string{"生成", "创建", "短链"}):
		return IntentCreated
	case pkg.Contains(intent, []string{"查询", "检查", "状态"}):
		return IntentQuery
	default:
		return IntentChat
	}
}
