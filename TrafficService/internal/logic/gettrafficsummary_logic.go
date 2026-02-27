package logic

import (
	"context"
	"errors"
	"strings"

	"trafficservice/internal/svc"
	"trafficservice/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTrafficSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetTrafficSummaryLogic 创建流量查询逻辑对象。
func NewGetTrafficSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTrafficSummaryLogic {
	return &GetTrafficSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetTrafficSummary 查询指定短码在时间区间内的汇总数据。
func (l *GetTrafficSummaryLogic) GetTrafficSummary(req *types.TrafficSummaryRequest) (*types.TrafficSummaryResponse, error) {
	if strings.TrimSpace(req.ShortCode) == "" {
		return nil, errors.New("shortCode is required")
	}

	if strings.TrimSpace(req.From) == "" || strings.TrimSpace(req.To) == "" {
		return nil, errors.New("from and to are required")
	}

	row, err := queryPVUV(l.ctx, l.svcCtx, req.ShortCode, req.From, req.To)
	if err != nil {
		return nil, err
	}

	trend, err := queryTrend(l.ctx, l.svcCtx, req.ShortCode, req.From, req.To)
	if err != nil {
		return nil, err
	}

	referrers, err := queryTopReferrers(l.ctx, l.svcCtx, req.ShortCode, req.From, req.To)
	if err != nil {
		return nil, err
	}

	return &types.TrafficSummaryResponse{
		ShortCode:    req.ShortCode,
		From:         req.From,
		To:           req.To,
		Pv:           row.PV,
		Uv:           row.UV,
		TopReferrers: referrers,
		Trend:        trend,
	}, nil
}
