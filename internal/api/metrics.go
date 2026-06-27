package api

import (
	"net/http"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricRequestDuration     *prometheus.HistogramVec
	metricRequestDurationOnce sync.Once
)

type requestDurationMetric struct {
	metric *prometheus.HistogramVec
}

func newRequestDurationMetric() *requestDurationMetric {
	metricRequestDurationOnce.Do(
		func() {
			o := prometheus.HistogramOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netbox_request_duration_seconds",
				Help: "Histogram of Netbox API round-trip latency, by " +
					"endpoint",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
			}
			metricRequestDuration = promauto.NewHistogramVec(
				o,
				[]string{"endpoint"},
			)
		},
	)
	return &requestDurationMetric{
		metric: metricRequestDuration,
	}
}

func (m *requestDurationMetric) Observe(endpoint string, d float64) {
	m.metric.WithLabelValues(endpoint).Observe(d)
}

var (
	metricRequestsTotal     *prometheus.CounterVec
	metricRequestsTotalOnce sync.Once
)

type requestsTotalMetric struct {
	metric *prometheus.CounterVec
}

func newRequestsTotalMetric() *requestsTotalMetric {
	metricRequestsTotalOnce.Do(
		func() {
			o := prometheus.CounterOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netbox_requests_total",
				Help: "Counter of Netbox API requests, by endpoint and HTTP " +
					"response code",
			}
			metricRequestsTotal = promauto.NewCounterVec(
				o,
				[]string{"endpoint", "code"},
			)
		},
	)
	return &requestsTotalMetric{
		metric: metricRequestsTotal,
	}
}

func (m *requestsTotalMetric) IncError(l string) {
	m.metric.WithLabelValues(l, "error").Inc()
}

func (m *requestsTotalMetric) IncStatusCode(l string, c int) {
	s := http.StatusText(c)
	if s == "" {
		s = "UNKNOWN"
	}
	m.metric.WithLabelValues(l, s).Inc()
}
