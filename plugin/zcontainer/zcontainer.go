package zcontainer

import (
	"net/http"

	"github.com/elvisNg/broccoliv2/mysql/zmysql"
	"github.com/elvisNg/broccoliv2/prometheus/zprometheus"

	"github.com/elvisNg/broccoliv2/httpclient/zhttpclient"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/sirupsen/logrus"

	"github.com/elvisNg/broccoliv2/config"
	"github.com/elvisNg/broccoliv2/mongo/zmongo"
	"github.com/elvisNg/broccoliv2/redis/zredis"
	tracing "github.com/elvisNg/broccoliv2/trace"
)

// Container 组件的容器访问接口
type Container interface {
	Init(appcfg *config.AppConf)
	Reload(appcfg *config.AppConf)
	GetRedisCli() zredis.Redis
	SetGoMicroClient(cli client.Client)
	GetGoMicroClient() client.Client
	GetLogger() *logrus.Logger
	GetAccessLogger() *logrus.Logger
	GetTracer() *tracing.TracerWrap
	SetServiceID(id string)
	GetServiceID() string
	SetHTTPHandler(h http.Handler)
	GetHTTPHandler() http.Handler
	SetGoMicroService(s micro.Service)
	GetGoMicroService() micro.Service
	GetMongo() zmongo.Mongo
	GetHttpClient() zhttpclient.HttpClient
	GetMysqlCli() zmysql.Mysql
	GetPrometheus() zprometheus.Prometheus
}
