// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"goZero/internal/config"
	"goZero/model"
	"goZero/sequence"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config        config.Config
	ShortUrlModel model.ShortUrlMapModel
	Sequence      sequence.Sequence // 序列生成器

	ShotUrlBlackList map[string]bool
	ShortDomain      string
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	conn := sqlx.NewMysql(c.ShortUrlDB.DSN)

	m := make(map[string]bool)
	//构造和黑名单
	for _, v := range c.ShortUrlBlackList {
		m[v] = true
	}

	// 初始化缓存连接
	redisConf := redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	}

	return &ServiceContext{
		Config:        c,
		ShortUrlModel: model.NewShortUrlMapModel(conn, c.RedisCache),
		//Sequence:      sequence.NewMysql(c.Sequence.DSN),
		Sequence: sequence.NewRedis(redisConf), // 发号器
		//Cache:            redis.MustNewRedis(redisConf), // 缓存层
		ShotUrlBlackList: m,
		ShortDomain:      c.ShortDomain,
	}

}
