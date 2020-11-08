package prom

import (
	"github.com/elvisNg/broccoliv2/config"
	"github.com/prometheus/client_golang/prometheus"
)

var prom *PromClient

// PromClient struct broccoli prometheus client & pub Business prometheus client
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

func InitClient(cfg *config.Prometheus) *PromClient {
	prom = &PromClient{
		innerClient: &InnerClient{
			HTTPServer: NewInner().withTimer("broccoli_http_server_duration", []string{"url"}).withCounter("broccoli_http_server_code", []string{"url", "err_code"}).withState("broccoli_http_server_state", []string{"url"}),
			RPCClient:  NewInner().withTimer("broccoli_rpc_client_duration", []string{"client_node", "server_name"}).withCounter("broccoli_rpc_client_code", []string{"client_node", "server_name", "err_code"}).withState("broccoli_rpc_client_state", []string{"server_name", "client_node"}),
			RPCServer:  NewInner().withTimer("broccoli_rpc_server_duration", []string{"server_node", "server_name"}).withCounter("broccoli_rpc_server_code", []string{"server_name", "err_code"}).withState("broccoli_rpc_server_state", []string{"server_name"}),
		},
		pHost: cfg.PullHost,
	}
	prom.pubClient = newPrometheusClient()
	return prom
}

func (prom *PromClient) GetPubCli() *PubClient {
	return prom.pubClient
}

func (prom *PromClient) GetInnerCli() *InnerClient {
	return prom.innerClient
}

// New creates a Prom instance.
func NewInner() *Prom {
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
func (p *Prom) Timing(name string, time int64, extra ...string) {
	label := append([]string{name}, extra...)
	if p.timer != nil {
		p.timer.WithLabelValues(label...).Observe(float64(time))
	}
}

// Incr increments one stat counter without sampling
func (p *Prom) Incr(name string, extra ...string) {
	label := append([]string{name}, extra...)
	if p.counter != nil {
		p.counter.WithLabelValues(label...).Inc()
	}
	/*if p.state != nil {
		p.state.WithLabelValues(label...).Inc()
	}*/
}

// Incr increments one stat gauge without sampling
func (p *Prom) StateIncr(name string, extra ...string) {
	label := append([]string{name}, extra...)
	if p.state != nil {
		p.state.WithLabelValues(label...).Inc()
	}
}

// Decr decrements one stat gauge without sampling
func (p *Prom) Decr(name string, extra ...string) {
	if p.state != nil {
		label := append([]string{name}, extra...)
		p.state.WithLabelValues(label...).Dec()
	}
}

// State set state
func (p *Prom) State(name string, v int64, extra ...string) {
	if p.state != nil {
		label := append([]string{name}, extra...)
		p.state.WithLabelValues(label...).Set(float64(v))
	}
}

// Add add count    v must > 0
func (p *Prom) Add(name string, v int64, extra ...string) {
	label := append([]string{name}, extra...)
	if p.counter != nil {
		p.counter.WithLabelValues(label...).Add(float64(v))
	}
	if p.state != nil {
		p.state.WithLabelValues(label...).Add(float64(v))
	}
}