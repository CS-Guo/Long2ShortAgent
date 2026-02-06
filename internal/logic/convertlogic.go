// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"goZero/internal/svc"
	"goZero/internal/types"
	"goZero/model"
	"goZero/pkg/base62"
	"goZero/pkg/connect"
	"goZero/pkg/md5"
	"goZero/pkg/url_tool"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ConvertLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConvertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConvertLogic {
	return &ConvertLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Convert 输入一个长链接，转为短链接
func (l *ConvertLogic) Convert(req *types.ConvertRequest) (resp *types.ConvertResponse, err error) {
	// 1. 校验长链接
	// 1.1 数据不能为nil
	// 在 handler 进行校验
	// 1.2 链接能通，不为非法链接，是个网址
	ok := connect.Get(req.LongUrl)
	if !ok {
		return nil, errors.New("无效链接")
	}
	// 1.3 判断之前是否转过
	longUrl := md5.Sum([]byte(req.LongUrl))
	// 查数据库
	u, err := l.svcCtx.ShortUrlModel.FindOneByMd5(l.ctx, sql.NullString{String: longUrl, Valid: true})
	if !errors.Is(err, sqlx.ErrNotFound) {
		if err != nil {
			return nil, fmt.Errorf("该链已经被转过: %v", u.Surl.String)
		}
		logx.Errorw("l.svcCtx.ShortUrlModel.FindOneByMd5 Failed", logx.LogField{Key: "err", Value: err})

		return &types.ConvertResponse{ShortUrl: fmt.Sprintf("该链接已经被转过了：%v", l.svcCtx.ShortDomain+u.Surl.String)}, err
	}
	// 1.4 输入的是完整的url , 避免循环转链接
	baseUrl, err := url_tool.BasePath(longUrl)
	// 拿到短链再去数据库中查，是非
	u, err = l.svcCtx.ShortUrlModel.FindOneBySurl(l.ctx, sql.NullString{String: baseUrl, Valid: true})
	if !errors.Is(err, sqlx.ErrNotFound) {
		if err != nil {
			return nil, fmt.Errorf("该链接以及是短链无法再转")
		}
		logx.Errorw("l.svcCtx.ShortUrlModel.FindOneBySurl failed", logx.LogField{Key: "err", Value: err})
		return nil, err
	}
	// 2. 取号
	var short string
	for {
		seq, err := l.svcCtx.Sequence.Next()
		if err != nil {
			logx.Errorw("l.svcCtx.Sequence.Next failed", logx.LogField{Key: "err", Value: err})
			return nil, err
		}
		//fmt.Println(seq)
		// 3. 号码转为短链
		// 3.1 安全性问题
		short = base62.Int2String(seq)

		// 判断是否为黑名单的
		if !l.svcCtx.ShotUrlBlackList[short] {
			break
		}
	}
	// 短域名避免特殊的词 和路由相关的,建立黑名单
	// 4. 存储长和短的映射关系
	shortMapLong := model.ShortUrlMap{
		Surl: sql.NullString{String: short, Valid: true},
		Lurl: sql.NullString{String: req.LongUrl, Valid: true},
		Md5:  sql.NullString{String: longUrl, Valid: true},
	}
	_, err = l.svcCtx.ShortUrlModel.Insert(l.ctx, &shortMapLong)
	if err != nil {
		logx.Errorw("l.svcCtx.ShortUrlModel.Insert failed", logx.LogField{Key: "err", Value: err})
		return nil, err
	}
	// 5. 返回响应
	ShortUrl := l.svcCtx.ShortDomain + short
	fmt.Println(ShortUrl)
	return &types.ConvertResponse{ShortUrl: ShortUrl}, nil
}
