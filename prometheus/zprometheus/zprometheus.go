package zprometheus

import (
	prom "github.com/elvisNg/broccoliv2/prometheus"
)

type Prometheus interface {
	GetPubCli() *prom.PubClient
	GetInnerCli() *prom.InnerClient
}
