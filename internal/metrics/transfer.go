package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Transfers Total /////////////////////////////////////////////////////////////

var (
	transfersTotalOnce   sync.Once
	transfersTotalMetric *prometheus.CounterVec
)

type transfersTotal struct {
	metric *prometheus.CounterVec
}

func newTransfersTotal(subsystem string) *transfersTotal {
	onceCounterVec(
		&transfersTotalOnce,
		&transfersTotalMetric,
		subsystem,
		"transfers_total",
		"Counter of outgoing zone transfers, by zone and kind (axfr, "+
			"ixfr_delta, ixfr_noop, ixfr_fallback, catalog_axfr, "+
			"catalog_ixfr_delta, catalog_ixfr_noop).",
		"zone",
		"kind",
	)
	return &transfersTotal{
		metric: transfersTotalMetric,
	}
}

func (t *transfersTotal) Inc(labels ...string) {
	t.metric.WithLabelValues(labels...).Inc()
}
