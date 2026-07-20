package poller

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

var (
	metricZonePollerErrorTotal     *prometheus.CounterVec
	metricZonePollerErrorTotalOnce sync.Once
)

func newZonePollerErrorsMetric() *prometheus.CounterVec {
	metricZonePollerErrorTotalOnce.Do(
		func() {
			opts := prometheus.CounterOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_zone_poller_error_total",
				Help:      "",
			}
			metricZonePollerErrorTotal = promauto.NewCounterVec(
				opts,
				[]string{"name"},
			)
		},
	)
	return metricZonePollerErrorTotal
}

var (
	metricZonePollerPollDuration     prometheus.Histogram
	metricZonePollerPollDurationOnce sync.Once
)

func newZonePollerPollDurationMetric() prometheus.Histogram {
	metricZonePollerPollDurationOnce.Do(
		func() {
			opts := prometheus.HistogramOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_zone_poller_poll_duration_seconds",
				Help:      "",
			}
			metricZonePollerPollDuration = promauto.NewHistogram(opts)
		},
	)
	return metricZonePollerPollDuration
}

var (
	metricZonePollerCyclesTotal     *prometheus.CounterVec
	metricZonePollerCyclesTotalOnce sync.Once
)

func newZonePollerCyclesTotalMetric() *prometheus.CounterVec {
	metricZonePollerCyclesTotalOnce.Do(
		func() {
			opts := prometheus.CounterOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_zone_poller_cycles_total",
				Help:      "",
			}
			metricZonePollerCyclesTotal = prometheus.NewCounterVec(
				opts,
				[]string{"name"},
			)
		},
	)
	return metricZonePollerCyclesTotal
}

var (
	metricZonePollerZoneSerial     *prometheus.GaugeVec
	metricZonePollerZoneSerialOnce sync.Once
)

func newZonePollerZoneSerialMetric() *prometheus.GaugeVec {
	metricZonePollerZoneSerialOnce.Do(
		func() {
			opts := prometheus.GaugeOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_zone_poller_zone_serial",
				Help:      "",
			}
			metricZonePollerZoneSerial = promauto.NewGaugeVec(
				opts,
				[]string{"zone"},
			)
		},
	)
	return metricZonePollerZoneSerial
}

var (
	metricZonePollerCacheSize     *prometheus.GaugeVec
	metricZonePollerCacheSizeOnce sync.Once
)

func newZonePollerCacheSizeMetric() *prometheus.GaugeVec {
	metricZonePollerCacheSizeOnce.Do(
		func() {
			opts := prometheus.GaugeOpts{
				Namespace: plugin.Namespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "netboxdns_zone_poller_cache_size",
				Help:      "",
			}
			metricZonePollerCacheSize = promauto.NewGaugeVec(
				opts,
				[]string{"zone"},
			)
		},
	)
	return metricZonePollerCacheSize
}
