// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"database/sql"
	"fmt"
	"goZero/internal/svc"
	"goZero/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ShowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewShowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShowLogic {
	return &ShowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ShowLogic) Show(req *types.ShowRequest) (resp *types.ShowResponse, err error) {
	// todo: add your logic here and delete this line
	// 1. 根据短链查长链接
	// 布隆过滤器防止缓存穿透
	// 1.1 从缓存中查
	ok, err := l.svcCtx.Filter.Exists([]byte(req.ShortUrl))

	if err != nil {
		logx.Errorw("l.svcCtx.Filter.Exists failed", logx.LogField{Key: "err", Value: err})
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	fmt.Println("开始查缓存DB")

	record, err := l.svcCtx.ShortUrlModel.FindOneBySurl(l.ctx, sql.NullString{String: req.ShortUrl, Valid: true})
	if err != nil {
		logx.Errorw("l.svcCtx.ShortUrlModel.FindOneBySurl failed", logx.LogField{Key: "err", Value: err})
		return nil, err
	}
	longUrl := record.Lurl.String

	return &types.ShowResponse{LongUrl: longUrl}, err
}
