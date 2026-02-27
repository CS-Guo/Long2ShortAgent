package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"goZero/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type clickEvent struct {
	EventID    string `json:"eventId"`
	TraceID    string `json:"traceId"`
	RequestID  string `json:"requestId"`
	OccurredAt string `json:"occurredAt"`
	ShortCode  string `json:"shortCode"`
	LongURL    string `json:"longUrl"`
	IP         string `json:"ip"`
	UA         string `json:"ua"`
	Referer    string `json:"referer"`
	Country    string `json:"country"`
	Province   string `json:"province"`
	City       string `json:"city"`
	DeviceType string `json:"deviceType"`
}

// emitClickEvent 将跳转访问事件写入 Redis Stream，供流量服务异步消费。
func emitClickEvent(ctx context.Context, svcCtx *svc.ServiceContext, shortCode, longURL, requestID, ip, ua, referer string) {
	stream := svcCtx.Config.Event.ClickStream
	if stream == "" {
		stream = "shortlink:click:event"
	}

	event := clickEvent{
		EventID:    randomID("evt"),
		TraceID:    randomID("trace"),
		RequestID:  requestID,
		OccurredAt: time.Now().Format(time.RFC3339),
		ShortCode:  shortCode,
		LongURL:    longURL,
		IP:         ip,
		UA:         ua,
		Referer:    referer,
		Country:    "",
		Province:   "",
		City:       "",
		DeviceType: detectDeviceType(ua),
	}

	_, err := svcCtx.RedisStore.XAddCtx(ctx, stream, false, "*", map[string]any{
		"eventId":    event.EventID,
		"traceId":    event.TraceID,
		"requestId":  event.RequestID,
		"occurredAt": event.OccurredAt,
		"shortCode":  event.ShortCode,
		"longUrl":    event.LongURL,
		"ip":         event.IP,
		"ua":         event.UA,
		"referer":    event.Referer,
		"country":    event.Country,
		"province":   event.Province,
		"city":       event.City,
		"deviceType": event.DeviceType,
	})
	if err != nil {
		logx.WithContext(ctx).Errorf("emit click event failed: %v", err)
	}
}

// detectDeviceType 根据 UA 粗略判断设备类型。
func detectDeviceType(ua string) string {
	u := stringsLower(ua)
	switch {
	case contains(u, "mobile"):
		return "mobile"
	case contains(u, "tablet") || contains(u, "ipad"):
		return "tablet"
	case contains(u, "bot") || contains(u, "spider") || contains(u, "crawler"):
		return "bot"
	case u == "":
		return "unknown"
	default:
		return "desktop"
	}
}

// randomID 生成事件/追踪 ID。
func randomID(prefix string) string {
	raw := make([]byte, 8)
	_, _ = rand.Read(raw)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(raw))
}

// stringsLower 是轻量 lower 实现，避免额外依赖。
func stringsLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 'A' && ch <= 'Z' {
			ch = ch + 32
		}
		b[i] = ch
	}
	return string(b)
}

// contains 是轻量子串判断函数。
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		matched := true
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
