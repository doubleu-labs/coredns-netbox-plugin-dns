package poller

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricViewPollerCyclesTotal     *prometheus.CounterVec
	metricViewPollerCyclesTotalOnce sync.Once
)

func newViewPollerCyclesTotalMetric() *prometheus.CounterVec {
	metricViewPollerCyclesTotalOnce.Do(
		func() {
			opts := prometheus.CounterOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_view_poller_cycles_total",
				Help:      "Number of times the view poller has completed a cycle.",
			}
			metricViewPollerCyclesTotal = prometheus.NewCounterVec(
				opts,
				[]string{"name"},
			)
		},
	)
	return metricViewPollerCyclesTotal
}

var (
	metricViewPollerPollDuration     prometheus.Histogram
	metricViewPollerPollDurationOnce sync.Once
)

func newViewPollerPollDurationMetric() prometheus.Histogram {
	metricViewPollerPollDurationOnce.Do(
		func() {
			opts := prometheus.HistogramOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_view_poller_poll_duration_seconds",
				Help:      "",
			}
			metricViewPollerPollDuration = prometheus.NewHistogram(opts)
		},
	)
	return metricViewPollerPollDuration
}

var (
	metricViewPollerErrorTotal     *prometheus.CounterVec
	metricViewPollerErrorTotalOnce sync.Once
)

func newViewPollerErrorsMetric() *prometheus.CounterVec {
	metricViewPollerErrorTotalOnce.Do(
		func() {
			opts := prometheus.CounterOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_view_poller_error_total",
				Help:      "",
			}
			metricViewPollerErrorTotal = prometheus.NewCounterVec(
				opts,
				[]string{"name"},
			)
		},
	)
	return metricViewPollerErrorTotal
}
