package zredis

import (
	"time"

	"github.com/elvisNg/broccoliv2/config"
	"github.com/go-redis/redis"
)

type Redis interface {
	Reload(cfg *config.Redis)
	//GetCli() *redis.Client
	ZGet(key string) *redis.StringCmd
	ZSet(key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	ZDel(key string) *redis.IntCmd
	ZIncr(key string) *redis.IntCmd
	ZTTL(key string) *redis.DurationCmd
	ZSetRange(key string, offset int64, value string) *redis.IntCmd
	ZSetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	ZExpire(key string, expiration time.Duration) *redis.BoolCmd
	ZExists(key string) *redis.IntCmd
}
