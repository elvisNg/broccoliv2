package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/elvisNg/broccoliv2/config"
	broccolictx "github.com/elvisNg/broccoliv2/context"
	"github.com/elvisNg/broccoliv2/errors"
	"github.com/elvisNg/broccoliv2/httpclient/zhttpclient"
	broccoliprometheus "github.com/elvisNg/broccoliv2/prometheus"
	tracing "github.com/elvisNg/broccoliv2/trace"
	"github.com/elvisNg/broccoliv2/utils"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tidwall/gjson"
)

const (
	defaultRetryCount  = 0
	defaultHTTPTimeout = 30 * 1000 * time.Millisecond
)

var (
	httpclientInstance = make(map[string]*Client)
	prom               **broccoliprometheus.Prom
	prometheus         = broccoliprometheus.NewProm()
)

type httpClientSettings struct {
	Transport       http.Transport
	RetryCount      uint32
	Timeout         time.Duration
	Hosts           []string
	TraceOnlyLogErr bool
}

type Client struct {
	client         *http.Client
	settings       httpClientSettings
	retrier        Retriable
	instanceName   string
	assertJSONPath string
	assertExpr     *vm.Program
	errCodePath    string
}

func InitHttpClientConf(conf map[string]config.HttpClientConf) {
	var tmpInstanceMap = make(map[string]*Client)
	for instanceName, httpClientConf := range conf {
		if v, ok := httpclientInstance[instanceName]; ok {
			v.client.CloseIdleConnections()
		}
		tmpInstanceMap[instanceName] = newClient(&httpClientConf)
	}
	httpclientInstance = tmpInstanceMap
	return
}

func InitHttpClientConfWithPorm(conf map[string]config.HttpClientConf, promClient **broccoliprometheus.Prom) {
	if promClient != nil {
		prom = promClient
		prometheus = *prom
	}
	var tmpInstanceMap = make(map[string]*Client)
	for instanceName, httpClientConf := range conf {
		if v, ok := httpclientInstance[instanceName]; ok {
			v.client.CloseIdleConnections()
		}
		tmpInstanceMap[instanceName] = newClient(&httpClientConf)
	}
	httpclientInstance = tmpInstanceMap
	return
}

func GetClient(instance string) (*Client, error) {
	v, ok := httpclientInstance[instance]
	if !ok {
		log.Printf("unknown instance: " + instance)
		return nil, errors.ECodeHttpClient.ParseErr("unknown instance: " + instance)
	}
	return v, nil
}

func (c *Client) GetHttpClient(instance string) (zhttpclient.Client, error) {
	clent, err := GetClient(instance)
	return clent, err
}

func DefaultClient() *Client {
	settings := httpClientSettings{
		Transport:       http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}},
		Timeout:         defaultHTTPTimeout,
		RetryCount:      defaultRetryCount,
		TraceOnlyLogErr: true,
	}

	client := Client{
		client: &http.Client{
			Transport: &settings.Transport,
			Timeout:   settings.Timeout,
		},
		settings: settings,
		retrier:  NewNoRetrier(),
	}
	return &client
}

func newClient(cfg *config.HttpClientConf) *Client {
	settings := httpClientSettings{
		Transport:       http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}},
		Timeout:         defaultHTTPTimeout,
		RetryCount:      defaultRetryCount,
		TraceOnlyLogErr: true,
	}

	settings.TraceOnlyLogErr = cfg.TraceOnlyLogErr

	if len(cfg.HostName) == 0 {
		panic("host_name不能为空...")
	}
	settings.Hosts = cfg.HostName

	if cfg.RetryCount != 0 {
		settings.RetryCount = cfg.RetryCount
	}

	if cfg.TimeOut != 0 {
		settings.Timeout = cfg.TimeOut * time.Millisecond
	}
	transport := http.Transport{}
	if cfg.IdleConnTimeout != 0 {
		transport.IdleConnTimeout = cfg.IdleConnTimeout * time.Millisecond
	}
	if cfg.MaxConnsPerHost != 0 {
		transport.MaxConnsPerHost = cfg.MaxConnsPerHost
	}
	if cfg.MaxIdleConns != 0 {
		transport.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxIdleConnsPerHost != 0 {
		transport.MaxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
	}

	if !cfg.InsecureSkipVerify && !utils.IsEmptyString(cfg.CaCertPath) {
		caCrt, err := ioutil.ReadFile(cfg.CaCertPath)
		if err != nil {
			panic(fmt.Sprintf("%v:读取证书文件错误:%v!", cfg.HostName, err.Error()))
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCrt)
		transport.TLSClientConfig = &tls.Config{RootCAs: pool}
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}

	settings.Transport = http.Transport{
		TLSClientConfig:     transport.TLSClientConfig,
		DisableKeepAlives:   cfg.DisableKeepAlives,
		MaxIdleConns:        transport.MaxIdleConns,
		MaxIdleConnsPerHost: transport.MaxIdleConnsPerHost,
		MaxConnsPerHost:     transport.MaxConnsPerHost,
		IdleConnTimeout:     transport.IdleConnTimeout,
	}

	retrier := NewNoRetrier()
	if cfg.RetryCount > 0 {
		retrier = NewRetrier(NewConstantBackoff(cfg.BackoffInterval*time.Millisecond, cfg.MaximumJitterInterval*time.Millisecond))
	}

	client := Client{
		client: &http.Client{
			Transport: &settings.Transport,
			Timeout:   settings.Timeout,
		},
		settings:     settings,
		retrier:      retrier,
		instanceName: cfg.InstanceName,
		errCodePath:  cfg.ErrCodePath,
	}

	// if len(cfg.AssertJSONPath) > 0 && len(cfg.AssertExpr) > 0 {
	// 	program, err := expr.Compile(cfg.AssertExpr)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	client.assertJSONPath = cfg.AssertJSONPath
	// 	client.assertExpr = program
	// }

	return &client
}

func httpClientStatus(url string, start time.Time, statusCode string) {
	prometheus.Timing("-", time.Since(start).Seconds(), url)
	prometheus.Incr("-", url, statusCode, "-")
	prometheus.StateIncr(url)
}

func (c *Client) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v%v", c.getRandomHost(), url), nil)
	if err != nil {
		return nil, errors.ECodeHttpClient.ParseErr("GET - request creation failed")
	}

	return c.do(ctx, request, headers)
}

func (c *Client) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v%v", c.getRandomHost(), url), body)
	if err != nil {
		return nil, errors.ECodeHttpClient.ParseErr("POST - request creation failed")
	}

	return c.do(ctx, request, headers)
}

func (c *Client) Put(ctx context.Context, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v%v", c.getRandomHost(), url), body)
	if err != nil {
		return nil, errors.ECodeHttpClient.ParseErr("PUT - request creation failed")
	}

	return c.do(ctx, request, headers)
}

func (c *Client) Patch(ctx context.Context, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%v%v", c.getRandomHost(), url), body)
	if err != nil {
		return nil, errors.ECodeHttpClient.ParseErr("PATCH - request creation failed")
	}

	return c.do(ctx, request, headers)
}

func (c *Client) Delete(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%v%v", c.getRandomHost(), url), nil)
	if err != nil {
		return nil, errors.ECodeHttpClient.ParseErr("DELETE - request creation failed")
	}

	return c.do(ctx, request, headers)
}

func (c *Client) getRandomHost() string {
	return c.settings.Hosts[rand.Intn(len(c.settings.Hosts))]
}

func (c *Client) do(ctx context.Context, request *http.Request, headers map[string]string) (rsp []byte, err error) {
	var now = time.Now()
	var statusCode string = "-"
	var errCode string = "-"
	if len(headers) > 0 {
		for k, v := range headers {
			request.Header.Add(k, v)
		}
	}

	//request.Close = true
	loger := broccolictx.ExtractLogger(ctx)
	tracer := tracing.NewTracerWrap(opentracing.GlobalTracer())
	defer func() {
		prometheus.Timing(c.instanceName, time.Since(now).Seconds(), request.URL.Path)
		prometheus.Incr(c.instanceName, request.URL.Path, statusCode, errCode)
	}()
	name := request.URL.RawPath
	ctx, span, _ := tracer.StartSpanFromContext(ctx, name)
	ext.SpanKindConsumer.Set(span)
	span.SetTag("httpclient request.method", request.Method)
	defer func() {
		if c.settings.TraceOnlyLogErr && err == nil {
			return
		}
		span.Finish()
		// FIXME: do we need this??
		// if prom != nil {
		// 	httpClientStatus(request.URL.Path, now, code)
		// }
	}()

	var bodyReader *bytes.Reader

	if request.Body != nil {
		reqData, err := ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		span.SetTag("httpclient request.body", string(reqData))
		bodyReader = bytes.NewReader(reqData)
		request.Body = ioutil.NopCloser(bodyReader) // prevents closing the body between retries
	}

	var response *http.Response

	for i := 0; i <= int(c.settings.RetryCount); i++ {
		if response != nil {
			response.Body.Close()
		}
		response, err = c.client.Do(request)
		if bodyReader != nil {
			// Reset the body reader after the request since at this point it's already read
			// Note that it's safe to ignore the error here since the 0,0 position is always valid
			_, _ = bodyReader.Seek(0, 0)
		}

		if err != nil {
			backoffTime := c.retrier.NextInterval(i)
			time.Sleep(backoffTime)
			continue
		}

		if response.StatusCode >= http.StatusInternalServerError {
			backoffTime := c.retrier.NextInterval(i)
			time.Sleep(backoffTime)
			continue
		}
		break
	}

	if response == nil {
		return nil, err
	}
	statusCode = strconv.Itoa(response.StatusCode)
	defer response.Body.Close()
	rspBody, err := ioutil.ReadAll(response.Body)
	span.SetTag("httpclient response.status", response.StatusCode)
	span.SetTag("httpclient response.body", string(rspBody))
	span.SetTag("httpclient response.error", err)
	if err != nil {
		loger.Errorf("ReadAll error:%+v", err)
		return nil, err
	}

	// if c.assertExpr != nil && strings.HasPrefix(response.Header.Get("Content-type"), "application/json") {
	// if c.assertExpr != nil {
	// 	ret, err := c.assertRespJSONValue(ctx, rspBody)
	// 	if err != nil {
	// 		errCode = err.Error()
	// 	} else {
	// 		if ret {
	// 			errCode = "OK"
	// 		} else {
	// 			errCode = "ERR"
	// 		}
	// 	}
	// }
	if c.errCodePath != "" {
		errCode = c.extractJSONErrCode(ctx, rspBody)
	}

	return rspBody, err
}

func (c *Client) extractJSONErrCode(ctx context.Context, rspBody []byte) string {
	jsonStr := string(rspBody)
	return gjson.Get(jsonStr, c.errCodePath).String()
}

func (c *Client) assertRespJSONValue(ctx context.Context, rspBody []byte) (bool, error) {
	logger := broccolictx.ExtractLogger(ctx)

	jsonStr := string(rspBody)

	value := gjson.Get(jsonStr, c.assertJSONPath)

	out, err := expr.Run(c.assertExpr, map[string]interface{}{"value": value})
	if err != nil {
		logger.Errorf("httpclient[%s] expr.Run error:%+v", c.instanceName, err)
		return false, fmt.Errorf("EXPR_ERROR")
	}
	ret, ok := out.(bool)
	if !ok {
		panic("httpclient expr ret is not bool")
	}

	return ret, nil
}
