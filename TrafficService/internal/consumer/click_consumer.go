package consumer

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"trafficservice/internal/svc"

	redislib "github.com/zeromicro/go-zero/core/stores/redis"
)

// EnsureGroup 确保 Redis Stream 消费组存在。
func EnsureGroup(ctx context.Context, svcCtx *svc.ServiceContext) error {
	stream := svcCtx.Config.Event.Stream
	group := svcCtx.Config.Event.Group

	if stream == "" {
		stream = "shortlink:click:event"
	}
	if group == "" {
		group = "traffic-service"
	}

	_, err := svcCtx.Redis.XGroupCreateMkStreamCtx(ctx, stream, group, "$")
	if err != nil {
		if contains(err.Error(), "BUSYGROUP") {
			return nil
		}
		return err
	}
	return nil
}

// Start 启动后台消费协程，持续拉取并处理点击事件。
func Start(ctx context.Context, svcCtx *svc.ServiceContext) {
	go func() {
		_ = EnsureGroup(ctx, svcCtx)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := consumeOnce(ctx, svcCtx); err != nil {
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
}

// consumeOnce 执行一次批量消费。
func consumeOnce(ctx context.Context, svcCtx *svc.ServiceContext) error {
	stream := svcCtx.Config.Event.Stream
	group := svcCtx.Config.Event.Group
	consumer := svcCtx.Config.Event.Consumer
	count := int64(svcCtx.Config.Event.BatchSize)
	blockMs := svcCtx.Config.Event.BlockMs

	if stream == "" {
		stream = "shortlink:click:event"
	}
	if group == "" {
		group = "traffic-service"
	}
	if consumer == "" {
		consumer = "traffic-worker-1"
	}
	if count <= 0 {
		count = 200
	}
	if blockMs <= 0 {
		blockMs = 2000
	}

	node, err := redislib.CreateBlockingNode(svcCtx.Redis)
	if err != nil {
		return err
	}
	defer node.Close()

	streams, err := svcCtx.Redis.XReadGroupCtx(
		ctx,
		node,
		group,
		consumer,
		count,
		time.Duration(blockMs)*time.Millisecond,
		false,
		stream,
		">",
	)
	if err != nil {
		if contains(err.Error(), "timeout") {
			return nil
		}
		if contains(err.Error(), "nil") {
			return nil
		}
		return err
	}

	for _, st := range streams {
		for _, msg := range st.Messages {
			if e := handleMessage(ctx, svcCtx, msg.Values); e == nil {
				_, _ = svcCtx.Redis.XAckCtx(ctx, stream, group, msg.ID)
			}
		}
	}

	return nil
}

// handleMessage 处理单条点击事件：落明细 + 更新日统计。
func handleMessage(ctx context.Context, svcCtx *svc.ServiceContext, values map[string]any) error {
	shortCode := asString(values["shortCode"])
	occurredAt := asString(values["occurredAt"])
	ip := asString(values["ip"])
	referer := asString(values["referer"])
	ua := asString(values["ua"])
	deviceType := asString(values["deviceType"])
	eventID := asString(values["eventId"])
	traceID := asString(values["traceId"])
	requestID := asString(values["requestId"])
	country := asString(values["country"])
	province := asString(values["province"])
	city := asString(values["city"])

	if shortCode == "" {
		return fmt.Errorf("shortCode is empty")
	}

	timeValue, err := time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		timeValue = time.Now()
	}

	if err := insertClickEvent(ctx, svcCtx, eventID, traceID, requestID, shortCode, ip, ua, referer, country, province, city, deviceType, timeValue); err != nil {
		if contains(err.Error(), "Duplicate entry") {
			return nil
		}
		return err
	}

	if err := upsertDailyStats(ctx, svcCtx, shortCode, timeValue, ip); err != nil {
		return err
	}

	return nil
}

// insertClickEvent 写入点击明细表。
func insertClickEvent(ctx context.Context, svcCtx *svc.ServiceContext, eventID, traceID, requestID, shortCode, ip, ua, referer, country, province, city, deviceType string, occurredAt time.Time) error {
	query := `
insert into short_url_click_event
  (event_id, trace_id, request_id, short_code, ip, ua, referer, country, province, city, device_type, occurred_at)
values
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := svcCtx.DB.ExecCtx(ctx, query,
		eventID,
		traceID,
		requestID,
		shortCode,
		ip,
		ua,
		referer,
		country,
		province,
		city,
		deviceType,
		occurredAt,
	)
	return err
}

// upsertDailyStats 聚合更新日维度 PV/UV。
func upsertDailyStats(ctx context.Context, svcCtx *svc.ServiceContext, shortCode string, occurredAt time.Time, ip string) error {
	dateKey := occurredAt.Format("2006-01-02")
	uvDelta := 0
	if firstVisit(ctx, svcCtx, shortCode, dateKey, ip) {
		uvDelta = 1
	}

	query := `
insert into short_url_daily_stat(stat_date, short_code, pv, uv)
values(?, ?, 1, ?)
on duplicate key update
  pv = pv + 1,
  uv = uv + values(uv)`

	_, err := svcCtx.DB.ExecCtx(ctx, query, dateKey, shortCode, uvDelta)
	return err
}

// firstVisit 用 Redis Set 估算当日 UV（按短码+日期+IP）。
func firstVisit(ctx context.Context, svcCtx *svc.ServiceContext, shortCode, dateKey, ip string) bool {
	if ip == "" {
		return false
	}

	uvKey := fmt.Sprintf("uv:%s:%s", shortCode, dateKey)
	exists, err := svcCtx.Redis.SismemberCtx(ctx, uvKey, ip)
	if err != nil {
		return false
	}
	if exists {
		return false
	}

	_, _ = svcCtx.Redis.SaddCtx(ctx, uvKey, ip)
	_ = svcCtx.Redis.ExpireCtx(ctx, uvKey, 3*24*3600)
	return true
}

// asString 将 Redis 读取出的动态类型统一转为字符串。
func asString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case fmt.Stringer:
		return val.String()
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case sql.NullString:
		if val.Valid {
			return val.String
		}
		return ""
	default:
		return ""
	}
}

// contains 轻量子串判断。
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
