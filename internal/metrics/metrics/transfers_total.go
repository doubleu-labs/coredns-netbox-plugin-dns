package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	transfersTotalOnce   sync.Once
	transfersTotalMetric *prometheus.CounterVec
)

type TransfersTotal struct {
	metric *prometheus.CounterVec
}

func NewTransfersTotal(subsystem string) *TransfersTotal {
	o := prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "transfers_total",
		Help: "counter of outgoing zone transfers, by zone and kind (axfr, " +
			"ixfr_delta, ixfr_noop, ixfr_fallback, catalog_axfr, " +
			"catalog_ixfr_delta, catalog_ixfr_noop)",
	}
	transfersTotalOnce.Do(
		func() {
			transfersTotalMetric = prometheus.NewCounterVec(
				o,
				[]string{"zone", "kind"},
			)
		},
	)
	return &TransfersTotal{
		metric: transfersTotalMetric,
	}
}

func (t *TransfersTotal) Inc(labels ...string) {
	t.metric.WithLabelValues(labels...).Inc()
}
