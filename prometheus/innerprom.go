package prom

import (
	"github.com/elvisNg/broccoliv2/config"
	"github.com/prometheus/client_golang/prometheus"
)

type Client struct {
	client *PromClient
}

// PromClient struct zeus prometheus client & pub Business prometheus client
type PromClient struct {
	innerClient *InnerClient
	pubClient   *PubClient
	pHost       string
}

type InnerClient struct {
	RPCClient *Prom
	//HTTPClient *Prom
	HTTPServer *Prom
	RPCServer  *Prom
}

// Prom struct info
type Prom struct {
	timer   *prometheus.HistogramVec
	counter *prometheus.CounterVec
	state   *prometheus.GaugeVec
}

func newClient() *Client {
	return &Client{
		client: nil,
	}
}

func InitClient(cfg *config.Prometheus) *Client {
	promClient := newClient()
	promClient.initClient(cfg)
	return promClient
}

func (promClient *Client) Enable() {
	if promClient == nil {
		return
	}
	promClient.client.pubClient.CacheMiss = New()
	promClient.client.pubClient.CacheHit = New()
	promClient.client.pubClient.BusinessInfoCount = New()
	promClient.client.pubClient.BusinessErrCount = New()
	promClient.client.pubClient.HTTPClient = NewInner().withTimer("zeus_http_client_duration_seconds", []string{"name", "url"}).withCounter("zeus_http_client_code", []string{"name", "url", "status_code", "err_code"}).withState("zeus_http_client_state", []string{"url"})
	promClient.client.pubClient.DbClient = NewInner().withTimer("zeus_db_client_duration_seconds", []string{"type", "method", "sql"}).withCounter("zeus_db_client_code", []string{"type", "method", "address", "code"}).withState("zeus_db_client_state", []string{"sql", "options"})
	promClient.client.pubClient.CacheClient = NewInner().withTimer("zeus_cache_client_duration_seconds", []string{"options", "key"}).withCounter("zeus_cache_client_code", []string{"options", "key", "msg"}).withState("zeus_cache_client_state", []string{"options", "key"})
	promClient.client.innerClient.HTTPServer = NewInner().withTimer("zeus_http_server_duration_seconds", []string{"url"}).withCounter("zeus_http_server_code", []string{"url", "err_code"}).withState("zeus_http_server_state", []string{"url"})
	promClient.client.innerClient.RPCClient = NewInner().withTimer("zeus_rpc_client_duration_seconds", []string{"server_name"}).withCounter("zeus_rpc_client_code", []string{"client_node", "server_name", "err_code"}).withState("zeus_rpc_client_state", []string{"server_name", "client_node"})
	promClient.client.innerClient.RPCServer = NewInner().withTimer("zeus_rpc_server_duration_seconds", []string{"server_name"}).withCounter("zeus_rpc_server_code", []string{"server_name", "err_code"})
}

func (promClient *Client) Disable() {
	if promClient == nil {
		return
	}
	promClient.client.pubClient.CacheMiss = nil
	promClient.client.pubClient.CacheHit = nil
	promClient.client.pubClient.BusinessInfoCount = nil
	promClient.client.pubClient.BusinessErrCount = nil
	promClient.client.pubClient.HTTPClient = nil
	promClient.client.pubClient.DbClient = nil
	promClient.client.pubClient.CacheClient = nil
	promClient.client.innerClient.HTTPServer = nil
	promClient.client.innerClient.RPCClient = nil
	promClient.client.innerClient.RPCServer = nil
}

func (promClient *Client) initClient(cfg *config.Prometheus) {
	prom := &PromClient{
		innerClient: &InnerClient{
			HTTPServer: NewInner().withTimer("zeus_http_server_duration_seconds", []string{"url"}).withCounter("zeus_http_server_code", []string{"url", "err_code"}).withState("zeus_http_server_state", []string{"url"}),
			RPCClient:  NewInner().withTimer("zeus_rpc_client_duration_seconds", []string{"server_name"}).withCounter("zeus_rpc_client_code", []string{"client_node", "server_name", "err_code"}).withState("zeus_rpc_client_state", []string{"server_name", "client_node"}),
			RPCServer:  NewInner().withTimer("zeus_rpc_server_duration_seconds", []string{"server_name"}).withCounter("zeus_rpc_server_code", []string{"server_name", "err_code"}),
		},
		pHost: cfg.PullHost,
	}
	prom.pubClient = newPrometheusClient()
	promClient.client = prom
}

func (promClient *Client) GetPubCli() *PubClient {
	return promClient.client.pubClient
}

func (promClient *Client) GetInnerCli() *InnerClient {
	return promClient.client.innerClient
}

// New creates a Prom instance.
func NewInner() *Prom {
	return &Prom{}
}

// NewProm New Prom
func NewProm() *Prom {
	return &Prom{}
}

// WithTimer with summary timer
func (p *Prom) withTimer(name string, labels []string) *Prom {
	if p == nil || p.timer != nil {
		return p
	}
	p.timer = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: name,
			Help: name,
		}, labels)
	prometheus.MustRegister(p.timer)
	return p
}

// WithCounter sets counter.
func (p *Prom) withCounter(name string, labels []string) *Prom {
	if p == nil || p.counter != nil {
		return p
	}
	p.counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: name,
		}, labels)
	prometheus.MustRegister(p.counter)
	return p
}

// WithState sets state.
func (p *Prom) withState(name string, labels []string) *Prom {
	if p == nil || p.state != nil {
		return p
	}
	p.state = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: name,
		}, labels)
	prometheus.MustRegister(p.state)
	return p
}

// Timing log timing information (in milliseconds) without sampling
func (p *Prom) Timing(name string, time float64, extra ...string) {
	if p == nil {
		return
	}
	label := append([]string{name}, extra...)
	if p.timer != nil {
		p.timer.WithLabelValues(label...).Observe(time)
	}
}

// Incr increments one stat counter without sampling
func (p *Prom) Incr(name string, extra ...string) {
	if p == nil {
		return
	}
	label := append([]string{name}, extra...)
	if p.counter != nil {
		p.counter.WithLabelValues(label...).Inc()
	}
}

// Incr increments one stat gauge without sampling
func (p *Prom) StateIncr(name string, extra ...string) {
	if p == nil {
		return
	}
	label := append([]string{name}, extra...)
	if p.state != nil {
		p.state.WithLabelValues(label...).Inc()
	}
}

// Decr decrements one stat gauge without sampling
func (p *Prom) Decr(name string, extra ...string) {
	if p == nil {
		return
	}
	if p.state != nil {
		label := append([]string{name}, extra...)
		p.state.WithLabelValues(label...).Dec()
	}
}

// State set state
func (p *Prom) State(name string, v int64, extra ...string) {
	if p == nil {
		return
	}
	if p.state != nil {
		label := append([]string{name}, extra...)
		p.state.WithLabelValues(label...).Set(float64(v))
	}
}

// Add add count    v must > 0
func (p *Prom) Add(name string, v int64, extra ...string) {
	if p == nil {
		return
	}
	label := append([]string{name}, extra...)
	if p.counter != nil {
		p.counter.WithLabelValues(label...).Add(float64(v))
	}
	if p.state != nil {
		p.state.WithLabelValues(label...).Add(float64(v))
	}
}
