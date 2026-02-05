package sequence

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Redis struct {
	conn *redis.Redis
}

func (r *Redis) Next() (uint64, error) {
	seq, err := r.conn.Incr("a")
	if err != nil {
		logx.Errorw("r.conn.Incr failed", logx.LogField{Key: "err", Value: err.Error()})
	}
	return uint64(seq), nil
}

func NewRedis(conf redis.RedisConf) Sequence {
	return &Redis{
		conn: redis.MustNewRedis(conf),
	}
}
