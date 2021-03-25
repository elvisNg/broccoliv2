package redisclient

import (
	"log"
	"sync"
	"time"

	broccoliprometheus "github.com/elvisNg/broccoliv2/prometheus"

	"github.com/elvisNg/broccoliv2/config"
	"github.com/go-redis/redis"
)

type ClusterClientClient struct {
	client *redis.ClusterClient
	rw     sync.RWMutex
}

func InitClusterClientWithProm(cfg *config.Redis, promClient **broccoliprometheus.Prom) *ClusterClientClient {
	rds := new(ClusterClientClient)
	prom = promClient
	prometheus = *prom
	rds.client = newRedisClusterClient(cfg)
	return rds
}

func InitClusterClient(cfg *config.Redis) *ClusterClientClient {
	rds := new(ClusterClientClient)
	rds.client = newRedisClusterClient(cfg)
	return rds
}

func newRedisClusterClient(cfg *config.Redis) *redis.ClusterClient {
	var clusterClient *redis.ClusterClient
	clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:       cfg.ClusterHost,
		Password:    cfg.Pwd,
		PoolSize:    cfg.PoolSize,
		IdleTimeout: time.Duration(cfg.ConnIdleTimeout) * time.Second,
	})
	if err := clusterClient.Ping().Err(); err != nil {
		log.Fatalf("[redis.newRedisClient] redis ping failed: %s\n", err.Error())
		return nil
	}
	log.Printf("[redis.newRedisClient] success \n")
	return clusterClient
}

func (rds *ClusterClientClient) Reload(cfg *config.Redis) {
	rds.rw.Lock()
	defer rds.rw.Unlock()
	if err := rds.client.Close(); err != nil {
		log.Printf("redis close failed: %s\n", err.Error())
		return
	}
	log.Printf("[redis.Reload] redisclient reload with new conf: %+v\n", cfg)
	rds.client = newRedisClusterClient(cfg)
}

func (rds *ClusterClientClient) GetCli() *redis.ClusterClient {
	rds.rw.RLock()
	defer rds.rw.RUnlock()
	return rds.client
}

func (rds *ClusterClientClient) release() {
	rds.rw.Lock()
	defer rds.rw.Unlock()
	if err := rds.client.Close(); err != nil {
		log.Printf("redis close failed: %s\n", err.Error())
		return
	}
}

func (rds *ClusterClientClient) ZGet(key string) *redis.StringCmd {
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

func (rds *ClusterClientClient) ZSet(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	//println("ClusterClientClient zsetting")
	getStartTime := time.Now()
	result := rds.client.Set(key, value, expiration)
	if result.Err() != nil {
		log.Printf("ClusterClientClient Set Error:", result.Err().Error())
		prometheus.Incr(redisSet, getKeyPerfix(key), result.Err().Error())
	} else {
		prometheus.Incr(redisSet, getKeyPerfix(key), OPTION_SUC)
	}
	prometheus.Timing(redisSet, time.Since(getStartTime).Seconds(), getKeyPerfix(key))
	prometheus.StateIncr(redisSet, getKeyPerfix(key))
	return result
}

func (rds *ClusterClientClient) ZDel(key string) *redis.IntCmd {
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

func (rds *ClusterClientClient) ZIncr(key string) *redis.IntCmd {
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

func (rds *ClusterClientClient) ZTTL(key string) *redis.DurationCmd {
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

func (rds *ClusterClientClient) ZSetRange(key string, offset int64, value string) *redis.IntCmd {
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

func (rds *ClusterClientClient) ZSetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
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

func (rds *ClusterClientClient) ZExpire(key string, expiration time.Duration) *redis.BoolCmd {
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

func (rds *ClusterClientClient) ZExists(key string) *redis.IntCmd {
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
