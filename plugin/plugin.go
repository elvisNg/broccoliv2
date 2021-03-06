package plugin

import (
	"log"
	"net/http"

	"github.com/elvisNg/broccoliv2/mysql/zmysql"
	"github.com/elvisNg/broccoliv2/prometheus/zprometheus"

	"github.com/elvisNg/broccoliv2/httpclient"
	"github.com/elvisNg/broccoliv2/httpclient/zhttpclient"

	"github.com/google/gops/agent"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/elvisNg/broccoliv2/config"
	broccolilog "github.com/elvisNg/broccoliv2/log"
	broccolimongo "github.com/elvisNg/broccoliv2/mongo"
	"github.com/elvisNg/broccoliv2/mongo/zmongo"
	broccolimysql "github.com/elvisNg/broccoliv2/mysql"
	broccoliprometheus "github.com/elvisNg/broccoliv2/prometheus"
	broccoliredis "github.com/elvisNg/broccoliv2/redis"
	"github.com/elvisNg/broccoliv2/redis/zredis"
	"github.com/elvisNg/broccoliv2/sequence"
	tracing "github.com/elvisNg/broccoliv2/trace"
	"github.com/elvisNg/broccoliv2/trace/zipkin"
)

// Container contain comm obj, impl zcontainer
type Container struct {
	serviceID     string
	appcfg        config.AppConf
	redis         zredis.Redis
	mongo         zmongo.Mongo
	mysql         zmysql.Mysql
	gomicroClient client.Client
	logger        *logrus.Logger
	accessLogger  *logrus.Logger
	tracer        *tracing.TracerWrap
	// http
	httpHandler http.Handler
	// gomicro grpc
	gomicroService micro.Service
	// httpclient
	httpClient zhttpclient.HttpClient
	prometheus zprometheus.Prometheus
	// dbPool          *sql.DB
	// transport       *http.Transport
	// svc             XUtil
	// mqProducer      *mq.MqProducer
}

func NewContainer() *Container {
	return &Container{}
}

func (c *Container) Init(appcfg *config.AppConf) {
	log.Println("[Container.Init] start")

	c.initLogger(&appcfg.LogConf)
	c.initAccessLogger(&appcfg.AccessLog)
	c.initTracer(&appcfg.Trace)
	c.initMongo(&appcfg.MongoDB)
	c.initPrometheus(&appcfg.Prometheus)
	if appcfg.Prometheus.Enable {
		c.initMysqlWithProm(&appcfg.Mysql, &c.prometheus.GetPubCli().DbClient)
		c.initRedisWithProm(&appcfg.Redis, &c.prometheus.GetPubCli().CacheClient)
		c.initHttpClientWithProm(appcfg.HttpClient, &c.prometheus.GetPubCli().HTTPClient)
	} else {
		c.initMysql(&appcfg.Mysql)
		c.initRedis(&appcfg.Redis)
		c.initHttpClient(appcfg.HttpClient)
	}
	c.initGoPS(&appcfg.GoPS)
	log.Println("[Container.Init] finish")
	c.appcfg = *appcfg
}

func (c *Container) Reload(appcfg *config.AppConf) {
	log.Println("[Container.Reload] start")
	if c.appcfg.Prometheus != appcfg.Prometheus {
		c.reloadPrometheus(&appcfg.Prometheus)
	}
	if c.appcfg.LogConf != appcfg.LogConf {
		c.reloadLogger(&appcfg.LogConf)
	}
	if c.appcfg.AccessLog.Conf != appcfg.AccessLog.Conf {
		c.reloadAccessLogger(&appcfg.AccessLog)
	}
	if c.appcfg.Trace != appcfg.Trace {
		c.reloadTracer(&appcfg.Trace)
	}
	if c.appcfg.MongoDB != appcfg.MongoDB {
		c.reloadMongo(&appcfg.MongoDB)
	}
	if c.appcfg.Mysql != appcfg.Mysql {
		c.reloadMysql(&appcfg.Mysql)
	}
	c.reloadRedis(&appcfg.Redis)
	c.initHttpClient(appcfg.HttpClient)
	if c.appcfg.GoPS != appcfg.GoPS {
		c.reloadGoPS(&appcfg.GoPS)
	}
	log.Println("[Container.Reload] finish")
	c.appcfg = *appcfg
}

// MysqlWithProm
func (c *Container) initRedisWithProm(cfg *config.Redis, promClient **broccoliprometheus.Prom) {
	if cfg.Enable {
		if cfg.ClusterEnable {
			c.redis = broccoliredis.InitClusterClient(cfg)
			if promClient != nil && cfg.Enable {
				c.redis = broccoliredis.InitClusterClientWithProm(cfg, promClient)
			}
		} else {
			c.redis = broccoliredis.InitClient(cfg)
			if promClient != nil && cfg.Enable {
				c.redis = broccoliredis.InitClientWithProm(cfg, promClient)
			}
		}
	}
}

// Redis
func (c *Container) initRedis(cfg *config.Redis) {
	if cfg.Enable {
		if cfg.ClusterEnable {
			c.redis = broccoliredis.InitClusterClient(cfg)
		} else {
			c.redis = broccoliredis.InitClient(cfg)
		}
	}
}

func (c *Container) reloadRedis(cfg *config.Redis) {
	if cfg.Enable {
		if c.redis != nil {
			c.redis.Reload(cfg)
		} else {
			c.redis = broccoliredis.InitClient(cfg)
		}
	} else if c.redis != nil {
		// 释放
		// c.redis.Release()
		c.redis = nil
	}
}

func (c *Container) GetRedisCli() zredis.Redis {
	return c.redis
}

// MysqlWithProm
func (c *Container) initMysqlWithProm(cfg *config.Mysql, promClient **broccoliprometheus.Prom) {
	if cfg.Enable {
		c.mysql = broccolimysql.InitClient(cfg)
	}
	if promClient != nil && cfg.Enable {
		c.mysql = broccolimysql.InitClientWithProm(cfg, promClient)
	}
}

func (c *Container) initMysql(cfg *config.Mysql) {
	if cfg.Enable {
		c.mysql = broccolimysql.InitClient(cfg)
	}

}

func (c *Container) reloadMysql(cfg *config.Mysql) {
	if cfg.Enable {
		if c.mysql != nil {
			c.mysql.Reload(cfg)
		} else {
			c.mysql = broccolimysql.InitClient(cfg)
		}
	} else if c.mysql != nil {
		// 释放
		// c.mysql.Release()
		c.mysql = nil
	}
}

func (c *Container) GetMysqlCli() zmysql.Mysql {
	return c.mysql
}

//Prometheus
func (c *Container) initPrometheus(cfg *config.Prometheus) {
	if cfg.Enable {
		c.prometheus = broccoliprometheus.InitClient(cfg)
	}
}
func (c *Container) GetPrometheus() zprometheus.Prometheus {
	return c.prometheus
}

//Prometheus
func (c *Container) reloadPrometheus(cfg *config.Prometheus) {
	if cfg.Enable {
		if c.prometheus.GetPubCli() != nil && c.prometheus.GetPubCli() != nil {
			c.prometheus.Enable()
		} else {
			c.prometheus = broccoliprometheus.InitClient(cfg)
		}
	} else if c.prometheus.GetPubCli() != nil && c.prometheus.GetPubCli() != nil {
		c.prometheus.Disable()
	}
}

// GoMicroClient
func (c *Container) SetGoMicroClient(cli client.Client) {
	c.gomicroClient = cli
}

func (c *Container) GetGoMicroClient() client.Client {
	return c.gomicroClient
}

// Logger
func (c *Container) initLogger(cfg *config.LogConf) {
	l, err := broccolilog.New(cfg)
	if err != nil {
		log.Println("initLogger err:", err)
		return
	}
	c.logger = l.Logger
}

func (c *Container) reloadLogger(cfg *config.LogConf) {
	c.initLogger(cfg)
}

func (c *Container) GetLogger() *logrus.Logger {
	return c.logger
}

// access logger
func (c *Container) initAccessLogger(cfg *config.AccessLog) {
	l, err := broccolilog.New(&cfg.Conf)
	if err != nil {
		log.Println("initAccessLogger err:", err)
		return
	}
	c.accessLogger = l.Logger
}

func (c *Container) reloadAccessLogger(cfg *config.AccessLog) {
	c.initAccessLogger(cfg)
}

func (c *Container) GetAccessLogger() *logrus.Logger {
	return c.accessLogger
}

// func (c *Container) SetDBPool(p *sql.DB) {
// 	c.dbPool = p
// }

// func (c *Container) GetDBPool() *sql.DB {
// 	return c.dbPool
// }

// func (c *Container) SetTransport(tr *http.Transport) {
// 	c.transport = tr
// }

// func (c *Container) GetTransport() *http.Transport {
// 	return c.transport
// }

// func (c *Container) SetSvcOptions(opt interface{}) {
// 	c.serviceOptions = opt
// }

// func (c *Container) GetSvcOptions() interface{} {
// 	return c.serviceOptions
// }

// func (c *Container) SetSvc(svc XUtil) {
// 	c.svc = svc
// }

// func (c *Container) GetSvc() XUtil {
// 	return c.svc
// }

// func (c *Container) SetMQProducer(p *mq.MqProducer) {
// 	c.mqProducer = p
// }

// func (c *Container) GetMQProducer() *mq.MqProducer {
// 	return c.mqProducer
// }

// func (c *Container) Release() {
// if c.redisPool != nil {
// 	c.redisPool.Close()
// }

// if c.dbPool != nil {
// 	c.dbPool.Close()
// }
// }

// Tracer
func (c *Container) initTracer(cfg *config.Trace) (err error) {
	err = zipkin.InitTracer(cfg)
	if err != nil {
		log.Println("initTracer err:", err)
		return
	}
	c.tracer = tracing.NewTracerWrap(opentracing.GlobalTracer())
	return
}

func (c *Container) reloadTracer(cfg *config.Trace) (err error) {
	return c.initTracer(cfg)
}

func (c *Container) GetTracer() *tracing.TracerWrap {
	return c.tracer
}

func (c *Container) SetServiceID(id string) {
	c.serviceID = id
	sequence.Load(id)
}

func (c *Container) GetServiceID() string {
	return c.serviceID
}

func (c *Container) SetHTTPHandler(h http.Handler) {
	c.httpHandler = h
}

func (c *Container) GetHTTPHandler() http.Handler {
	return c.httpHandler
}

func (c *Container) SetGoMicroService(s micro.Service) {
	c.gomicroService = s
}

func (c *Container) GetGoMicroService() micro.Service {
	return c.gomicroService
}

// Mongo
func (c *Container) initMongo(cfg *config.MongoDB) {
	var err error
	if cfg.Enable {
		broccolimongo.InitDefalut(cfg)
		c.mongo, err = broccolimongo.DefaultClient()
		if err != nil {
			log.Println("mgoc.DefaultClient err: ", err)
			return
		}
	}
}

func (c *Container) reloadMongo(cfg *config.MongoDB) {
	var err error
	if cfg.Enable {
		if c.mongo != nil {
			broccolimongo.ReloadDefault(cfg)
			c.mongo, err = broccolimongo.DefaultClient()
			if err != nil {
				log.Println("mgoc.DefaultClient err: ", err)
				return
			}
		} else {
			c.initMongo(cfg)
		}
	} else if c.mongo != nil {
		// 释放
		broccolimongo.DefaultClientRelease()
		c.mongo = nil
	}
}

func (c *Container) GetMongo() zmongo.Mongo {
	return c.mongo
}

// httpclient
func (c *Container) initHttpClient(conf map[string]config.HttpClientConf) {
	flag := c.appcfg.Trace.OnlyLogErr
	for _, v := range conf {
		v.TraceOnlyLogErr = flag
	}
	httpclient.InitHttpClientConf(conf)
	c.httpClient = httpclient.DefaultClient()
}

//initHttpClientWithProm
func (c *Container) initHttpClientWithProm(conf map[string]config.HttpClientConf, promClient **broccoliprometheus.Prom) {
	flag := c.appcfg.Trace.OnlyLogErr
	for _, v := range conf {
		v.TraceOnlyLogErr = flag
	}

	if promClient != nil {
		httpclient.InitHttpClientConfWithPorm(conf, promClient)
	} else {
		httpclient.InitHttpClientConf(conf)
	}

	c.httpClient = httpclient.DefaultClient()

}

func (c *Container) GetHttpClient() zhttpclient.HttpClient {
	return c.httpClient
}

func (c *Container) initGoPS(conf *config.GoPS) {
	// 启动进程监控
	if conf.Enable {
		log.Println("run gops agent")
		if err := agent.Listen(agent.Options{
			Addr:            conf.Addr,
			ConfigDir:       conf.ConfigDir,
			ShutdownCleanup: conf.ShutdownCleanup,
		}); err != nil {
			log.Println("run gops agent err:", err)
		}
	}
}

func (c *Container) reloadGoPS(conf *config.GoPS) {
	agent.Close()
	c.initGoPS(conf)
}

//GetMysql
func (c *Container) GetMysql() zmysql.Mysql {
	return c.mysql
}
