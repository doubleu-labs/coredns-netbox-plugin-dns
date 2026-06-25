package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	RequestsTotal   *requestsTotal
	RequestDuration *requestsDuration

	NetboxRequestsTotal   *netboxRequestsTotal
	NetboxRequestDuration *netboxRequestDuration

	TransfersTotal *transfersTotal

	PollCyclesTotal *pollCyclesTotal
	PollDuration    pollDuration

	ZoneSerial     *zoneSerial
	CacheSnapshots *cacheSnapshots
	CatalogMembers *catalogMembers
}

func NewMetrics(subsystem string) *Metrics {
	return &Metrics{
		RequestsTotal:         newRequestsTotal(subsystem),
		RequestDuration:       newRequestsDuration(subsystem),
		NetboxRequestsTotal:   newNetboxRequestsTotal(subsystem),
		NetboxRequestDuration: newNetboxRequestDuration(subsystem),
		TransfersTotal:        newTransfersTotal(subsystem),
		PollCyclesTotal:       newPollCyclesTotal(subsystem),
		PollDuration:          newPollDuration(subsystem),
		ZoneSerial:            newZoneSerial(subsystem),
		CacheSnapshots:        newCacheSnapshots(subsystem),
		CatalogMembers:        newCatalogMembers(subsystem),
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
