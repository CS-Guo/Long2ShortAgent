// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"database/sql"
	"fmt"
	"goZero/internal/svc"
	"goZero/internal/types"
	"net/http"
	"time"

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

// Show 根据短码查询长链接，并在命中后异步上报点击事件。
func (l *ShowLogic) Show(req *types.ShowRequest, httpReq *http.Request) (resp *types.ShowResponse, err error) {
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

	// 过期短链返回 404（与未命中行为一致，不暴露内部状态）。
	if record.ExpireAt.Valid && !record.ExpireAt.Time.After(time.Now()) {
		return nil, nil
	}

	longUrl := record.Lurl.String

	// 读取请求 ID，便于跨服务追踪
	requestID := httpReq.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = httpReq.Header.Get("X-Request-ID")
	}

	// 上报点击事件到 Redis Stream（失败不影响跳转主链路）
	emitClickEvent(
		l.ctx,
		l.svcCtx,
		req.ShortUrl,
		longUrl,
		requestID,
		httpReq.RemoteAddr,
		httpReq.UserAgent(),
		httpReq.Referer(),
	)

	return &types.ShowResponse{LongUrl: longUrl}, err
}
