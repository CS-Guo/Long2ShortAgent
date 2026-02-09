// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	RedisCache cache.CacheConf // 缓存配置

	ShortUrlDB struct {
		DSN string
	}

	Sequence struct {
		DSN string
	}

	BaseStr string

	ShortUrlBlackList []string

	ShortDomain string

	Redis struct {
		Host string
		Type string
		Pass string
	}
}
