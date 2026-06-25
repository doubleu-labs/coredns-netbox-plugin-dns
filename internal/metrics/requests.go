package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

// Requests Total //////////////////////////////////////////////////////////////

var (
	requestsTotalOnce   sync.Once
	requestsTotalMetric *prometheus.CounterVec
)

type requestsTotal struct {
	metric *prometheus.CounterVec
}

func newRequestsTotal(subsystem string) *requestsTotal {
	onceCounterVec(
		&requestsTotalOnce,
		&requestsTotalMetric,
		subsystem,
		"requests_total",
		"Counter of DNS request handled by netboxdns plugin by zone and "+
			"rcode.",
		"zone",
		"rcode",
	)
	return &requestsTotal{
		metric: requestsTotalMetric,
	}
}

func (r *requestsTotal) Inc(zone string, rcode string) {
	r.metric.WithLabelValues(zone, rcode).Inc()
}

// Request Duration ////////////////////////////////////////////////////////////

var (
	requestDurationOnce   sync.Once
	requestDurationMetric *prometheus.HistogramVec
)

type requestsDuration struct {
	metric *prometheus.HistogramVec
}

func newRequestsDuration(subsystem string) *requestsDuration {
	onceHistogramVec(
		&requestDurationOnce,
		&requestDurationMetric,
		subsystem,
		"request_duration_seconds",
		"Histogram of DNS request handling latency in netboxdns plugin.",
		plugin.TimeBuckets,
		"zone",
	)
	return &requestsDuration{
		metric: requestDurationMetric,
	}
}

func (r *requestsDuration) Observe(zone string, d float64) {
	r.metric.WithLabelValues(zone).Observe(d)
}

// Netbox Requests Total ///////////////////////////////////////////////////////

var (
	netboxRequestsTotalOnce   sync.Once
	netboxRequestsTotalMetric *prometheus.CounterVec
)

type netboxRequestsTotal struct {
	metric *prometheus.CounterVec
}

func newNetboxRequestsTotal(subsystem string) *netboxRequestsTotal {
	onceCounterVec(
		&netboxRequestsTotalOnce,
		&netboxRequestsTotalMetric,
		subsystem,
		"netbox_requests_total",
		"Counter of HTTP requests to the NetBox API, by endpoint and HTTP "+
			`code("error" when the round-trip itself failed).`,
		"endpoint",
		"code",
	)
	return &netboxRequestsTotal{
		metric: netboxRequestsTotalMetric,
	}
}

func (n *netboxRequestsTotal) inc(labels ...string) {
	n.metric.WithLabelValues(labels...).Inc()
}

// Netbox Request Duration /////////////////////////////////////////////////////

var (
	netboxRequestDurationOnce   sync.Once
	netboxRequestDurationMetric *prometheus.HistogramVec
)

type netboxRequestDuration struct {
	metric *prometheus.HistogramVec
}

func newNetboxRequestDuration(subsystem string) *netboxRequestDuration {
	onceHistogramVec(
		&netboxRequestDurationOnce,
		&netboxRequestDurationMetric,
		subsystem,
		"netbox_request_duration_seconds",
		"Histogram of NetBox API round-trip latency, by endpoint.",
		prometheus.ExponentialBuckets(0.001, 2, 14),
		"endpoint",
	)
	return &netboxRequestDuration{
		metric: netboxRequestDurationMetric,
	}
}

func (n *netboxRequestDuration) observe(endpoint string, d float64) {
	n.metric.WithLabelValues(endpoint).Observe(d)
}
