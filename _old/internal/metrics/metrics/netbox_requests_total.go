package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	netboxRequestsTotalOnce   sync.Once
	netboxRequestsTotalMetric *prometheus.CounterVec
)

type NetboxRequestsTotal struct {
	metric *prometheus.CounterVec
}

func NewNetboxRequestsTotal(subsystem string) *NetboxRequestsTotal {
	o := prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "netbox_requests_total",
		Help: "counter of HTTP requests to the NetBox API, by endpoint and " +
			"HTTP code",
	}
	netboxRequestsTotalOnce.Do(
		func() {
			netboxRequestsTotalMetric = promauto.NewCounterVec(
				o,
				[]string{"endpoint", "code"},
			)
		},
	)
	return &NetboxRequestsTotal{
		metric: netboxRequestsTotalMetric,
	}
}

func (n *NetboxRequestsTotal) Inc(labels ...string) {
	n.metric.WithLabelValues(labels...).Inc()
}
