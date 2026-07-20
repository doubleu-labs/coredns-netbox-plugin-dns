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
			opts := prometheus.HistogramOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netbox_request_duration_seconds",
				Help: "Histogram of Netbox API round-trip latency, by " +
					"endpoint",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
			}
			metricRequestDuration = promauto.NewHistogramVec(
				opts,
				[]string{"endpoint"},
			)
		},
	)
	return &requestDurationMetric{
		metric: metricRequestDuration,
	}
}

func (m *requestDurationMetric) observe(endpoint string, duration float64) {
	m.metric.WithLabelValues(endpoint).Observe(duration)
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
			opts := prometheus.CounterOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netbox_requests_total",
				Help: "Counter of Netbox API requests, by endpoint and HTTP " +
					"response code",
			}
			metricRequestsTotal = promauto.NewCounterVec(
				opts,
				[]string{"endpoint", "code"},
			)
		},
	)
	return &requestsTotalMetric{
		metric: metricRequestsTotal,
	}
}

func (m *requestsTotalMetric) incError(label string) {
	m.metric.WithLabelValues(label, "error").Inc()
}

func (m *requestsTotalMetric) incStatusCode(label string, code int) {
	status := http.StatusText(code)
	if status == "" {
		status = "UNKNOWN"
	}
	m.metric.WithLabelValues(label, status).Inc()
}
