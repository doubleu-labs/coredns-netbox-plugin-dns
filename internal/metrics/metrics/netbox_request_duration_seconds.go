package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	netboxRequestDurationOnce   sync.Once
	netboxRequestDurationMetric *prometheus.HistogramVec
)

type NetboxRequestDuration struct {
	metric *prometheus.HistogramVec
}

func NewNetboxRequestDuration(subsystem string) *NetboxRequestDuration {
	o := prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "netbox_request_duration_seconds",
		Help:      "histogram of NetBox API round-trip latency, by endpoint",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14),
	}
	netboxRequestDurationOnce.Do(
		func() {
			netboxRequestDurationMetric = promauto.NewHistogramVec(
				o,
				[]string{"endpoint"},
			)
		},
	)
	return &NetboxRequestDuration{
		metric: netboxRequestDurationMetric,
	}
}

func (n *NetboxRequestDuration) Observe(endpoint string, d float64) {
	n.metric.WithLabelValues(endpoint).Observe(d)
}
