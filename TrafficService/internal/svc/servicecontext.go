package svc

import (
	"trafficservice/internal/config"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis
}

// NewServiceContext 初始化流量服务依赖（MySQL + Redis）。
func NewServiceContext(c config.Config) *ServiceContext {
	redisClient := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})

	return &ServiceContext{
		Config: c,
		DB:     sqlx.NewMysql(c.ShortUrlDB.DSN),
		Redis:  redisClient,
	}
}
