package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec

	NetboxRequestsTotal   *prometheus.CounterVec
	NetboxRequestDuration *prometheus.HistogramVec

	TransfersTotal *prometheus.CounterVec

	PollCyclesTotal *prometheus.CounterVec
	PollDuration    prometheus.Histogram

	ZoneSerial     *prometheus.GaugeVec
	CacheSnapshots *prometheus.GaugeVec
	CatalogMembers *prometheus.GaugeVec
}

var (
	requestsTotalOnce sync.Once
	requestsTotal     *prometheus.CounterVec

	requestDurationOnce sync.Once
	requestDuration     *prometheus.HistogramVec

	netboxRequestsTotalOnce sync.Once
	netboxRequestsTotal     *prometheus.CounterVec

	netboxRequestDurationOnce sync.Once
	netboxRequestDuration     *prometheus.HistogramVec

	transfersTotalOnce sync.Once
	transfersTotal     *prometheus.CounterVec

	pollCyclesTotalOnce sync.Once
	pollCyclesTotal     *prometheus.CounterVec

	pollDurationOnce sync.Once
	pollDuration     prometheus.Histogram

	zoneSerialOnce sync.Once
	zoneSerial     *prometheus.GaugeVec

	cacheSnapshotsOnce sync.Once
	cacheSnapshots     *prometheus.GaugeVec

	catalogMembersOnce sync.Once
	catalogMembers     *prometheus.GaugeVec
)

func NewMetrics(s string) *Metrics {
	onceCounterVec(
		&requestsTotalOnce,
		&requestsTotal,
		s,
		"requests_total",
		"Counter of DNS request handled by netboxdns plugin by zone and rcode.",
		"zone",
		"rcode",
	)
	onceHistogramVec(
		&requestDurationOnce,
		&requestDuration,
		s,
		"request_duration_seconds",
		"Histogram of DNS request handling latency in netboxdns plugin.",
		plugin.TimeBuckets,
		"zone",
	)
	onceCounterVec(
		&netboxRequestsTotalOnce,
		&netboxRequestsTotal,
		s,
		"netbox_requests_total",
		"Counter of HTTP requests to the NetBox API, by endpoint and HTTP code "+
			"(\"error\" when the round-trip itself failed).",
		"endpoint",
		"code",
	)
	onceHistogramVec(
		&netboxRequestDurationOnce,
		&netboxRequestDuration,
		s,
		"netbox_request_duration_seconds",
		"Histogram of NetBox API round-trip latency, by endpoint.",
		prometheus.ExponentialBuckets(0.001, 2, 14),
		"endpoint",
	)
	onceCounterVec(
		&transfersTotalOnce,
		&transfersTotal,
		s,
		"transfers_total",
		"Counter of outgoing zone transfers, by zone and kind (axfr, ixfr_delta, "+
			"ixfr_noop, ixfr_fallback, catalog_axfr, catalog_ixfr_delta, "+
			"catalog_ixfr_noop).",
		"zone",
		"kind",
	)
	onceCounterVec(
		&pollCyclesTotalOnce,
		&pollCyclesTotal,
		s,
		"poll_cycles_total",
		"Counter of full poll-cycle executions.",
		"result",
	)
	onceHistogram(
		&pollDurationOnce,
		&pollDuration,
		s,
		"poll_duration_seconds",
		"Histogram of poll-cycle execution latency.",
		prometheus.ExponentialBuckets(0.01, 2, 14),
	)
	onceGaugeVec(
		&zoneSerialOnce,
		&zoneSerial,
		s,
		"zone_serial",
		"Current serial number of the zone.",
		"zone",
	)
	onceGaugeVec(
		&cacheSnapshotsOnce,
		&cacheSnapshots,
		s,
		"cache_snapshots",
		"Number of cache snapshots taken.",
		"zone",
	)
	onceGaugeVec(
		&catalogMembersOnce,
		&catalogMembers,
		s,
		"catalog_members",
		"Number of members in the catalog zone.",
		"catalog",
	)

	return &Metrics{
		RequestsTotal:         requestsTotal,
		RequestDuration:       requestDuration,
		NetboxRequestsTotal:   netboxRequestsTotal,
		NetboxRequestDuration: netboxRequestDuration,
		TransfersTotal:        transfersTotal,
		PollCyclesTotal:       pollCyclesTotal,
		PollDuration:          pollDuration,
		ZoneSerial:            zoneSerial,
		CacheSnapshots:        cacheSnapshots,
		CatalogMembers:        catalogMembers,
	}
}

func onceCounterVec(
	once *sync.Once,
	metric **prometheus.CounterVec,
	s string,
	n string,
	h string,
	l ...string,
) {
	once.Do(
		func() {
			*metric = promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: plugin.Namespace,
					Subsystem: s,
					Name:      n,
					Help:      h,
				},
				l,
			)
		},
	)
}

func onceHistogram(
	once *sync.Once,
	metric *prometheus.Histogram,
	s string,
	n string,
	h string,
	b []float64,
) {
	once.Do(
		func() {
			*metric = promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: plugin.Namespace,
					Subsystem: s,
					Name:      n,
					Help:      h,
					Buckets:   b,
				},
			)
		},
	)
}

func onceHistogramVec(
	once *sync.Once,
	metric **prometheus.HistogramVec,
	s string,
	n string,
	h string,
	b []float64,
	l ...string,
) {
	once.Do(
		func() {
			*metric = promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: plugin.Namespace,
					Subsystem: s,
					Name:      n,
					Help:      h,
					Buckets:   b,
				},
				l,
			)
		},
	)
}

func onceGaugeVec(
	once *sync.Once,
	metric **prometheus.GaugeVec,
	s string,
	n string,
	h string,
	l ...string,
) {
	once.Do(
		func() {
			*metric = promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: plugin.Namespace,
					Subsystem: s,
					Name:      n,
					Help:      h,
				},
				l,
			)
		},
	)
}
