package logic

import (
	"context"
	"errors"
	"strings"

	"goZero/internal/svc"
	"goZero/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type InternalCreateShortLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewInternalCreateShortLinkLogic 创建内部创建短链逻辑对象。
func NewInternalCreateShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InternalCreateShortLinkLogic {
	return &InternalCreateShortLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Create 复用 Convert 逻辑创建短链，并补充 shortCode 字段返回给调用方。
func (l *InternalCreateShortLinkLogic) Create(req *types.InternalCreateShortLinkRequest) (*types.InternalCreateShortLinkResponse, error) {
	if strings.TrimSpace(req.LongUrl) == "" {
		return nil, errors.New("longUrl is required")
	}

	// 复用公开接口的核心转换逻辑，避免两套创建逻辑分叉
	resp, err := NewConvertLogic(l.ctx, l.svcCtx).Convert(&types.ConvertRequest{LongUrl: req.LongUrl})
	if err != nil {
		return nil, err
	}

	// 从完整短链中提取 shortCode
	shortURL := resp.ShortUrl
	shortCode := strings.TrimPrefix(shortURL, l.svcCtx.ShortDomain)
	shortCode = strings.TrimPrefix(shortCode, "https://")
	shortCode = strings.TrimPrefix(shortCode, "http://")

	if idx := strings.LastIndex(shortCode, "/"); idx >= 0 {
		shortCode = shortCode[idx+1:]
	}

	return &types.InternalCreateShortLinkResponse{
		ShortUrl:  shortURL,
		ShortCode: shortCode,
	}, nil
}
