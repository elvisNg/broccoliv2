package prom

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Prom struct info
type PromPub struct {
	timer   map[string]*prometheus.HistogramVec
	counter map[string]*prometheus.CounterVec
	state   map[string]*prometheus.GaugeVec
}

// PubClient Business prometheus client
type PubClient struct {
	BusinessErrCount  *PromPub
	BusinessInfoCount *PromPub
	CacheHit          *PromPub
	CacheMiss         *PromPub
	DbClient          *Prom
	CacheClient       *Prom
	HTTPClient        *Prom
}

func newPrometheusClient() *PubClient {
	promPubClient := &PubClient{
		DbClient:          NewInner().withTimer("zeus_db_client_duration_seconds", []string{"type", "method", "sql"}).withCounter("zeus_db_client_code", []string{"type", "method", "address", "code"}).withState("zeus_db_client_state", []string{"sql", "options"}),
		CacheClient:       NewInner().withTimer("zeus_cache_client_duration_seconds", []string{"options", "key"}).withCounter("zeus_cache_client_code", []string{"options", "key", "msg"}).withState("zeus_cache_client_state", []string{"options", "key"}),
		BusinessErrCount:  New(),
		BusinessInfoCount: New(),
		CacheHit:          New(),
		CacheMiss:         New(),
		HTTPClient:        NewInner().withTimer("zeus_http_client_duration_seconds", []string{"name", "url"}).withCounter("zeus_http_client_code", []string{"name", "url", "status_code", "err_code"}).withState("zeus_http_client_state", []string{"url"}),
	}
	log.Printf("[prometheus.newPrometheusClient] success \n")
	return promPubClient
}

// New creates a Prom pub instance.
func New() *PromPub {
	return &PromPub{
		timer:   make(map[string]*prometheus.HistogramVec),
		counter: make(map[string]*prometheus.CounterVec),
		state:   make(map[string]*prometheus.GaugeVec),
	}
}

// PromPub WithPubTimer with summary timer
func (pb *PromPub) WithPubTimer(name string, labels []string) *Prom {
	if _, ok := pb.timer[name]; ok {
		return &Prom{timer: pb.timer[name]}
	} else {
		p := &Prom{
			timer: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name: name,
					Help: name,
				}, labels)}
		prometheus.MustRegister(p.timer)
		pb.timer[name] = p.timer
		return p
	}
}

//PromPub WithPubCounter sets counter.
func (pb *PromPub) WithPubCounter(name string, labels []string) *Prom {
	if _, ok := pb.counter[name]; ok {
		return &Prom{counter: pb.counter[name]}
	} else {
		p := &Prom{
			counter: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: name,
					Help: name,
				}, labels)}
		prometheus.MustRegister(p.counter)
		pb.counter[name] = p.counter
		return p
	}
}

//PromPub WithPubState sets counter.
func (pb *PromPub) WithPubState(name string, labels []string) *Prom {
	if _, ok := pb.state[name]; ok {
		return &Prom{state: pb.state[name]}
	} else {
		p := &Prom{
			state: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: name,
					Help: name,
				}, labels)}
		prometheus.MustRegister(p.state)
		pb.state[name] = p.state
		return p
	}
}

func (p *Prom) DbStatus(sql string, option string, start time.Time, affRow string, errMsg string) {
	p.Timing(sql, time.Since(start).Seconds(), affRow)
	p.Incr(sql, errMsg)
	p.StateIncr(sql, option)
}

func (p *Prom) CacheStatus(key string, option string, start time.Time, errMsg string) {
	p.Timing(option, time.Since(start).Seconds(), key)
	p.Incr(option, key, errMsg)
	p.StateIncr(option, key)
}

func (p *Prom) HttpClientStatus(params string, url string, start time.Time, errCode string, statusCode string) {
	p.Timing(url, time.Since(start).Seconds())
	p.Incr(url, statusCode)
	p.StateIncr(url)
}
