package redisclient

import (
	"log"
	"strings"
	"sync"
	"time"

	broccoliprometheus "github.com/elvisNg/broccoliv2/prometheus"

	"github.com/elvisNg/broccoliv2/config"
	"github.com/go-redis/redis"
)

const (
	redisGet    = "redis:get"
	redisSet    = "redis:set"
	redisDel    = "redis:del"
	redisTtl    = "redis:ttl"
	redisIncr   = "redis:incr"
	redisSetNx  = "redis:setnx"
	redisExpire = "redis:expire"
	redisExist  = "redis:exist"
	OPTION_SUC  = "success"
)

var (
	prom       **broccoliprometheus.Prom
	prometheus = broccoliprometheus.NewProm()
)

type Client struct {
	client *redis.Client
	rw     sync.RWMutex
}

func InitClientWithProm(cfg *config.Redis, promClient **broccoliprometheus.Prom) *Client {
	rds := new(Client)

	if promClient != nil {
		prom = promClient
		prometheus = *prom
	}
	rds.client = newRedisClient(cfg)
	return rds
}

func InitClient(cfg *config.Redis) *Client {
	rds := new(Client)
	rds.client = newRedisClient(cfg)
	return rds
}

func newRedisClient(cfg *config.Redis) *redis.Client {
	var client *redis.Client
	if cfg.SentinelHost != "" {
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.SentinelMastername,
			SentinelAddrs: strings.Split(cfg.SentinelHost, ","),
			Password:      cfg.Pwd,
			PoolSize:      cfg.PoolSize,
			IdleTimeout:   time.Duration(cfg.ConnIdleTimeout) * time.Second,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:        cfg.Host,
			Password:    cfg.Pwd,
			PoolSize:    cfg.PoolSize,
			IdleTimeout: time.Duration(cfg.ConnIdleTimeout) * time.Second,
		})
	}
	if err := client.Ping().Err(); err != nil {
		log.Fatalf("[redis.newRedisClient] redis ping failed: %s\n", err.Error())
		return nil
	}
	log.Printf("[redis.newRedisClient] success \n")
	return client
}

func (rds *Client) Reload(cfg *config.Redis) {
	rds.rw.Lock()
	defer rds.rw.Unlock()
	if err := rds.client.Close(); err != nil {
		log.Printf("redis close failed: %s\n", err.Error())
		return
	}
	log.Printf("[redis.Reload] redisclient reload with new conf: %+v\n", cfg)
	rds.client = newRedisClient(cfg)
}

func (rds *Client) GetCli() *redis.Client {
	rds.rw.RLock()
	defer rds.rw.RUnlock()
	return rds.client
}

func (rds *Client) release() {
	rds.rw.Lock()
	defer rds.rw.Unlock()
	if err := rds.client.Close(); err != nil {
		log.Printf("redis close failed: %s\n", err.Error())
		return
	}
}

func (rds *Client) ZGet(key string) *redis.StringCmd {
	getStartTime := time.Now()
	result := rds.client.Get(key)
	if result.Err() != nil {
		prometheus.Incr(redisGet, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisGet, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisGet, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisGet, getKeyPerfix(key))
	return result
}

func (rds *Client) ZSet(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	getStartTime := time.Now()

	result := rds.client.Set(key, value, expiration)
	if result.Err() != nil {
		prometheus.Incr(redisSet, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisSet, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisSet, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisSet, getKeyPerfix(key))
	return result
}

func (rds *Client) ZDel(key string) *redis.IntCmd {
	getStartTime := time.Now()
	result := rds.client.Del(key)
	if result.Err() != nil {
		prometheus.Incr(redisDel, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisDel, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisDel, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisDel, getKeyPerfix(key))
	return result
}

func (rds *Client) ZIncr(key string) *redis.IntCmd {
	getStartTime := time.Now()
	result := rds.client.Incr(key)
	if result.Err() != nil {
		prometheus.Incr(redisIncr, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisIncr, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisIncr, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisIncr, getKeyPerfix(key))
	return result
}

func (rds *Client) ZTTL(key string) *redis.DurationCmd {
	getStartTime := time.Now()
	result := rds.client.TTL(key)
	if result.Err() != nil {
		prometheus.Incr(redisTtl, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisTtl, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisTtl, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisTtl, getKeyPerfix(key))
	return result
}

func (rds *Client) ZSetRange(key string, offset int64, value string) *redis.IntCmd {
	getStartTime := time.Now()
	result := rds.client.SetRange(key, offset, value)
	if result.Err() != nil {
		prometheus.Incr(redisTtl, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisTtl, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisTtl, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisTtl, getKeyPerfix(key))
	return result
}

func (rds *Client) ZSetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	getStartTime := time.Now()
	result := rds.client.SetNX(key, value, expiration)
	if result.Err() != nil {
		prometheus.Incr(redisSetNx, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisSetNx, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisSetNx, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisSetNx, getKeyPerfix(key))
	return result
}

func (rds *Client) ZExpire(key string, expiration time.Duration) *redis.BoolCmd {
	getStartTime := time.Now()
	result := rds.client.Expire(key, expiration)
	if result.Err() != nil {
		prometheus.Incr(redisExpire, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisExpire, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisExpire, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisExpire, getKeyPerfix(key))
	return result
}

func (rds *Client) ZExists(key string) *redis.IntCmd {
	getStartTime := time.Now()
	result := rds.client.Exists(key)
	if result.Err() != nil {
		prometheus.Incr(redisExist, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisExist, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisExist, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisExist, getKeyPerfix(key))
	return result
}

func getKeyPerfix(key string) string {
	var keys []string
	keys = strings.Split(key, ":")
	if len(keys) > 0 {
		keys = keys[:len(keys)-1]
	}
	return strings.Join(keys, "")
}
