package logic

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"goZero/internal/svc"
	"goZero/internal/types"
	"goZero/model"
	"goZero/pkg/md5"

	"github.com/zeromicro/go-zero/core/logx"
)

type InternalUpdateShortLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewInternalUpdateShortLinkLogic 创建内部修改短链逻辑对象。
func NewInternalUpdateShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InternalUpdateShortLinkLogic {
	return &InternalUpdateShortLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Update 根据 shortCode 更新短链配置（长链接与过期时间可独立修改）。
func (l *InternalUpdateShortLinkLogic) Update(req *types.InternalUpdateShortLinkRequest) (*types.InternalUpdateShortLinkResponse, error) {
	if strings.TrimSpace(req.ShortCode) == "" {
		return nil, errors.New("shortCode is required")
	}

	newLongURL := strings.TrimSpace(req.LongUrl)
	newExpireAtRaw := strings.TrimSpace(req.ExpireAt)
	if newLongURL == "" && newExpireAtRaw == "" {
		return nil, errors.New("at least one field to update is required: longUrl or expireAt")
	}

	// 1) 先按短码查询原记录
	record, err := l.svcCtx.ShortUrlModel.FindOneBySurl(l.ctx, sql.NullString{String: req.ShortCode, Valid: true})
	if err != nil {
		return nil, err
	}

	// 2) 计算更新后的值（未传的字段保持原值）
	updatedLongURL := record.Lurl
	updatedMd5 := record.Md5
	if newLongURL != "" {
		updatedLongURL = sql.NullString{String: newLongURL, Valid: true}
		updatedMd5 = sql.NullString{String: md5.Sum([]byte(newLongURL)), Valid: true}
	}

	updatedExpireAt := record.ExpireAt
	if newExpireAtRaw != "" {
		parsedExpireAt, parseErr := parseExpireAt(newExpireAtRaw)
		if parseErr != nil {
			return nil, parseErr
		}
		updatedExpireAt = parsedExpireAt
	}

	// 3) 构造更新对象（保留主键与创建信息）
	updated := model.ShortUrlMap{
		Id:       record.Id,
		CreateAt: record.CreateAt,
		CreateBy: record.CreateBy,
		IsDel:    record.IsDel,
		Lurl:     updatedLongURL,
		Md5:      updatedMd5,
		Surl:     record.Surl,
		ExpireAt: updatedExpireAt,
	}

	if err := l.svcCtx.ShortUrlModel.Update(l.ctx, &updated); err != nil {
		return nil, err
	}

	// 4) 返回修改结果
	return &types.InternalUpdateShortLinkResponse{
		Updated:   true,
		ShortCode: req.ShortCode,
		LongUrl:   updatedLongURL.String,
		ExpireAt:  formatNullTimeRFC3339(updatedExpireAt),
	}, nil
}

// parseExpireAt 解析过期时间，支持 RFC3339 / yyyy-mm-dd hh:mm:ss / yyyy-mm-dd。
func parseExpireAt(raw string) (sql.NullTime, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return sql.NullTime{}, errors.New("expireAt is empty")
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err != nil {
			continue
		}

		// 日期格式默认取当天 23:59:59，避免用户意图与系统默认不一致。
		if layout == "2006-01-02" {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, parsed.Location())
		}

		return sql.NullTime{Time: parsed, Valid: true}, nil
	}

	return sql.NullTime{}, errors.New("invalid expireAt format, supported: RFC3339 / 2006-01-02 15:04:05 / 2006-01-02")
}

// formatNullTimeRFC3339 将 NullTime 格式化为 RFC3339 字符串。
func formatNullTimeRFC3339(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}
