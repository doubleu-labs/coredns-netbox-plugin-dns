package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestsTotalOnce   sync.Once
	requestsTotalMetric *prometheus.CounterVec
)

type RequestsTotal struct {
	metric *prometheus.CounterVec
}

func NewRequestsTotal(subsystem string) *RequestsTotal {
	o := prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "requests_total",
		Help: "counter of DNS request handled by netboxdns plugin by zone " +
			"and rcode",
	}
	requestsTotalOnce.Do(
		func() {
			requestsTotalMetric = prometheus.NewCounterVec(
				o,
				[]string{"zone", "rcode"},
			)
		},
	)
	return &RequestsTotal{
		metric: requestsTotalMetric,
	}
}

func (r *RequestsTotal) Inc(zone string, rcode string) {
	r.metric.WithLabelValues(zone, rcode).Inc()
}
