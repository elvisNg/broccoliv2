package zgomicro

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	zredis "github.com/elvisNg/broccoliv2/redis"

	"github.com/micro/go-micro/client"
	gmerrors "github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/server"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"

	broccolictx "github.com/elvisNg/broccoliv2/context"
	"github.com/elvisNg/broccoliv2/engine"
	broccolierrors "github.com/elvisNg/broccoliv2/errors"
	"github.com/elvisNg/broccoliv2/utils"
	"github.com/micro/go-micro/metadata"
)

var fm sync.Map

type validator interface {
	Validate() error
}

const (
	// log file.
	_source = "source"
	// container ID.
	_instanceID = "instance_id"
	// uniq ID from trace.
	_tid = "traceId"
	// appsName.
	_caller = "caller"
)

func GenerateServerLogWrap(ng engine.Engine) func(fn server.HandlerFunc) server.HandlerFunc {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) (err error) {
			logger := ng.GetContainer().GetLogger()
			var errcode string
			now := time.Now()
			l := logger.WithFields(logrus.Fields{"tag": "gomicro-serverlogwrap"})
			c := broccolictx.GMClientToContext(ctx, ng.GetContainer().GetGoMicroClient())

			defer func() {
				if panicErr := recover(); panicErr != nil {
					l.Errorf("ProcePanic: (%+v), stack: (%s)", panicErr, string(utils.Stack(3))) //string(debug.Stack())
					err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: "Internal error panic", Status: "ng.hanlder"}
					return
				}
			}()

			///////// tracer begin
			name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
			cfg, err := ng.GetConfiger()
			if err != nil {
				l.Error(err)
				err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: "ng.GetConfiger"}
				return
			}

			tracer := ng.GetContainer().GetTracer()
			if tracer == nil {
				err = fmt.Errorf("tracer is nil")
				l.Error(err)
				err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: ""}
				return
			}
			spnctx, span, err := tracer.StartSpanFromContext(c, name)
			if err != nil {
				l.Error(err)
				err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: ""}
				return
			}
			defer func() {
				if cfg.Get().Trace.OnlyLogErr && err == nil {
					return
				}
				span.Finish()
			}()
			body, _ := utils.Marshal(req.Body())
			span.SetTag("grpc server receive", string(body))
			///////// tracer finish
			tracerID := tracer.GetTraceID(spnctx)
			l = l.WithFields(logrus.Fields{
				_tid:        tracerID,
				_instanceID: getHostIP(),
				_source:     funcName(2),
				_caller:     ng.GetContainer().GetServiceID(),
			})
			c = broccolictx.LoggerToContext(spnctx, l)

			if v, ok := req.Body().(validator); ok && v != nil {
				if err = v.Validate(); err != nil {
					broccoliErr := broccolierrors.New(broccolierrors.ECodeInvalidParams, err.Error(), "validator.Validate")
					status := broccoliErr.Cause
					if !strings.HasPrefix(status, tracerID+"@") {
						status = tracerID + "@" + broccoliErr.Cause
					}
					err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccoliErr.ErrCode), Detail: broccoliErr.ErrMsg, Status: status}
					return
				}
			}
			if ng.GetContainer().GetMongo() != nil {
				c = broccolictx.MongoToContext(c, ng.GetContainer().GetMongo())
			}
			if ng.GetContainer().GetHttpClient() != nil {
				c = broccolictx.HttpclientToContext(c, ng.GetContainer().GetHttpClient())
			}
			if ng.GetContainer().GetMysqlCli() != nil {
				c = broccolictx.MysqlToContext(c, ng.GetContainer().GetMysqlCli().GetCli())
			}
			if ng.GetContainer().GetPrometheus() != nil {
				c = broccolictx.PrometheusToContext(c, ng.GetContainer().GetPrometheus().GetPubCli())
				if ng.GetContainer().GetRedisCli() != nil {
					c = broccolictx.RedisWithPromToContext(c, ng.GetContainer().GetRedisCli())
				}
			} else if ng.GetContainer().GetRedisCli() != nil {
				c = broccolictx.RedisToContext(c, (ng.GetContainer().GetRedisCli()).(*zredis.Client).GetCli())
			}

			defer func() {
				if cfg.Get().Prometheus.Enable {
					prom := ng.GetContainer().GetPrometheus().GetInnerCli()
					prom.RPCServer.Timing(name, int64(time.Since(now)/time.Millisecond), ng.GetContainer().GetServiceID())
					if errcode != "" {
						prom.RPCServer.Incr(name, errcode)
					} else {
						prom.RPCServer.Incr(name, strconv.Itoa(0))
					}
					prom.RPCServer.StateIncr(ng.GetContainer().GetServiceID())
				}
			}()
			err = fn(c, req, rsp)
			if err != nil && !utils.IsBlank(reflect.ValueOf(err)) {
				span.SetTag("grpc server answer error", err)
				// broccoli错误包装为gomicro错误
				var broccoliErr *broccolierrors.Error
				var gmErr *gmerrors.Error
				if errors.As(err, &broccoliErr) {
					serviceID := broccoliErr.ServiceID
					if utils.IsEmptyString(serviceID) {
						serviceID = ng.GetContainer().GetServiceID()
					}
					status := broccoliErr.Cause
					if !strings.HasPrefix(status, tracerID+"@") {
						status = tracerID + "@" + broccoliErr.Cause
					}
					gmErr = &gmerrors.Error{Id: serviceID, Code: int32(broccoliErr.ErrCode), Detail: broccoliErr.ErrMsg, Status: status}

					// 防止go-micro grpc 将小于errcode小于0的错误转换成 internal error
					// 对于非broccoli grpc调用，不做处理（grpc-gateway调用，直接返回负数，否http访问会得不到正确的错误码）
					if isbroccoliRpc(ctx) {
						if gmErr.Code < 0 {
							gmErr.Code = -gmErr.Code
							gmErr.Detail = "-@" + gmErr.Detail
						}
					}
					errcode = strconv.Itoa(int(broccoliErr.ErrCode))
					err = gmErr

					return
				}
				if errors.As(err, &gmErr) {
					err = gmErr
					return
				}

				err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: tracerID + "@" + err.Error()}
				return
			}
			err = nil
			rspRaw, _ := utils.Marshal(rsp)
			span.SetTag("grpc server answer", string(rspRaw))
			return
		}
	}
}

func GenerateClientLogWrap(ng engine.Engine) func(c client.Client) client.Client {
	return func(c client.Client) client.Client {
		return &clientLogWrap{
			Client: c,
			ng:     ng,
		}
	}
}

func GenerateClientWrapTest(ng engine.Engine) func(c client.Client) client.Client {
	return func(c client.Client) client.Client {
		return &clientWrapTest{
			Client: c,
		}
	}
}

func newClientLogWrap(c client.Client) client.Client {
	return &clientLogWrap{
		Client: c,
	}
}

type clientLogWrap struct {
	client.Client
	ng engine.Engine
}

func (l *clientLogWrap) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) (err error) {
	var now = time.Now()
	var errcode string
	logger := broccolictx.ExtractLogger(ctx)
	ng := l.ng

	///////// tracer begin
	name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())

	cfg, err := ng.GetConfiger()
	if err != nil {
		logger.Error(err)
		errcode = string(broccolierrors.ECodeSystem)
		err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: "ng.GetConfiger"}
		return
	}

	tracer := ng.GetContainer().GetTracer()
	if tracer == nil {
		err = fmt.Errorf("tracer is nil")
		logger.Error(err)
		errcode = string(broccolierrors.ECodeSystem)
		err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: ""}
		return
	}
	spnctx, span, err := tracer.StartSpanFromContext(ctx, name)
	if err != nil {
		logger.Error(err)
		errcode = string(broccolierrors.ECodeSystem)
		err = &gmerrors.Error{Id: ng.GetContainer().GetServiceID(), Code: int32(broccolierrors.ECodeSystem), Detail: err.Error(), Status: ""}
		return
	}
	defer func() {
		if cfg.Get().Trace.OnlyLogErr && err == nil {
			return
		}
		span.Finish()
	}()
	ext.SpanKindRPCClient.Set(span)
	body, _ := utils.Marshal(req.Body())
	span.SetTag("grpc client call", string(body))
	///////// tracer finish

	//broccoli rpc 调用标识
	ctx = broccoliFlagToContext(ctx)
	err = l.Client.Call(ctx, req, rsp, opts...)
	if err != nil {
		// gomicro错误解包为broccoli错误
		var gmErr *gmerrors.Error
		if errors.As(err, &gmErr) {
			if gmErr != nil {
				span.SetTag("grpc client receive error", gmErr)
				errcode = strconv.Itoa(int(gmErr.Code))
				// 根据Detail 判断错误码是否负数，将其还原
				strs := strings.SplitN(gmErr.Detail, "@", 2)
				if len(strs) == 2 && strs[0] == "-" {
					gmErr.Code = -gmErr.Code
					gmErr.Detail = strs[1]
				}
				broccoliErr := broccolierrors.New(broccolierrors.ErrorCode(gmErr.Code), gmErr.Detail, gmErr.Status)
				broccoliErr.ServiceID = gmErr.Id
				if utils.IsEmptyString(broccoliErr.ServiceID) && l.ng != nil {
					broccoliErr.ServiceID = l.ng.GetContainer().GetServiceID()
				}

				broccoliErr.TracerID = tracer.GetTraceID(spnctx)
				err = broccoliErr
				return
			}
			err = nil
		}
	}
	defer func() {
		if cfg.Get().Prometheus.Enable {
			prom := ng.GetContainer().GetPrometheus().GetInnerCli()
			prom.RPCClient.Timing(name, int64(time.Since(now)/time.Millisecond), ng.GetContainer().GetServiceID())
			if errcode != "" {
				prom.RPCClient.Incr(name, ng.GetContainer().GetServiceID(), errcode)
			} else {
				prom.RPCClient.Incr(name, ng.GetContainer().GetServiceID(), strconv.Itoa(0))
			}
			//mark rpc tracing
			prom.RPCClient.StateIncr(name, ng.GetContainer().GetServiceID())
		}
	}()
	rspRaw, _ := utils.Marshal(rsp)
	span.SetTag("grpc client receive", string(rspRaw))
	return
}

type clientWrapTest struct {
	client.Client
}

func (l *clientWrapTest) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) (err error) {
	broccolictx.ExtractLogger(ctx).Debug("clientWrapTest")
	err = l.Client.Call(ctx, req, rsp, opts...)
	return
}

func getHostIP() (orghost string) {
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				orghost = ipnet.IP.String()
			}

		}
	}
	return
}

// funcName get func name.
func funcName(skip int) (name string) {
	if pc, _, lineNo, ok := runtime.Caller(skip); ok {
		if v, ok := fm.Load(pc); ok {
			name = v.(string)
		} else {
			name = runtime.FuncForPC(pc).Name() + ":" + strconv.FormatInt(int64(lineNo), 10)
			fm.Store(pc, name)
		}
	}
	return
}

func broccoliFlagToContext(ctx context.Context) context.Context {
	if md, ok := metadata.FromContext(ctx); ok {
		md["broccoli-rpc-flag"] = ""
		//map修改直接生效，不需要重设
		//ctx = metadata.NewContext(ctx, md)
	} else {
		ctx = metadata.NewContext(ctx, metadata.Metadata{"broccoli-rpc-flag": ""})
	}
	return ctx
}

func isbroccoliRpc(ctx context.Context) bool {
	if md, ok := metadata.FromContext(ctx); ok {
		if _, ok := md["broccoli-rpc-flag"]; ok {
			return true
		}
	}
	return false
}
