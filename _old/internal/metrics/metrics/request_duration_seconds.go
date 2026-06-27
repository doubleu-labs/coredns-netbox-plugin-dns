package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestDurationOnce   sync.Once
	requestDurationMetric *prometheus.HistogramVec
)

type RequestsDuration struct {
	metric *prometheus.HistogramVec
}

func NewRequestsDuration(subsystem string) *RequestsDuration {
	o := prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "request_duration_seconds",
		Help: "histogram of DNS request handling latency in netboxdns " +
			"plugin",
		Buckets: plugin.TimeBuckets,
	}
	requestDurationOnce.Do(
		func() {
			requestDurationMetric = promauto.NewHistogramVec(
				o,
				[]string{"zone"},
			)
		},
	)
	return &RequestsDuration{
		metric: requestDurationMetric,
	}
}

func (r *RequestsDuration) Observe(zone string, d float64) {
	r.metric.WithLabelValues(zone).Observe(d)
}
