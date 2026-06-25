package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Poll Cycles /////////////////////////////////////////////////////////////////

var (
	pollCyclesTotalOnce   sync.Once
	pollCyclesTotalMetric *prometheus.CounterVec
)

type pollCyclesTotal struct {
	metric *prometheus.CounterVec
}

func newPollCyclesTotal(subsystem string) *pollCyclesTotal {
	onceCounterVec(
		&pollCyclesTotalOnce,
		&pollCyclesTotalMetric,
		subsystem,
		"poll_cycles_total",
		"Counter of full poll-cycle executions.",
		"result",
	)
	return &pollCyclesTotal{
		metric: pollCyclesTotalMetric,
	}
}

func (p *pollCyclesTotal) Inc(result string) {
	p.metric.WithLabelValues(result).Inc()
}

// Poll Duration ///////////////////////////////////////////////////////////////

var (
	pollDurationOnce   sync.Once
	pollDurationMetric prometheus.Histogram
)

type pollDuration struct {
	metric prometheus.Histogram
}

func newPollDuration(subsystem string) pollDuration {
	onceHistogram(
		&pollDurationOnce,
		&pollDurationMetric,
		subsystem,
		"poll_duration_seconds",
		"Histogram of poll-cycle execution latency.",
		prometheus.ExponentialBuckets(0.01, 2, 14),
	)
	return pollDuration{
		metric: pollDurationMetric,
	}
}

func (p *pollDuration) Observe(d float64) {
	p.metric.Observe(d)
}

// Zone Serial /////////////////////////////////////////////////////////////////

var (
	zoneSerialOnce   sync.Once
	zoneSerialMetric *prometheus.GaugeVec
)

type zoneSerial struct {
	metric *prometheus.GaugeVec
}

func newZoneSerial(subsystem string) *zoneSerial {
	onceGaugeVec(
		&zoneSerialOnce,
		&zoneSerialMetric,
		subsystem,
		"zone_serial",
		"Current serial number of the zone.",
		"zone",
	)
	return &zoneSerial{
		metric: zoneSerialMetric,
	}
}

func (z *zoneSerial) Set(zone string, serial uint32) {
	z.metric.WithLabelValues(zone).Set(float64(serial))
}

// Cache Snapshots /////////////////////////////////////////////////////////////

var (
	cacheSnapshotsOnce   sync.Once
	cacheSnapshotsMetric *prometheus.GaugeVec
)

type cacheSnapshots struct {
	metric *prometheus.GaugeVec
}

func newCacheSnapshots(subsystem string) *cacheSnapshots {
	onceGaugeVec(
		&cacheSnapshotsOnce,
		&cacheSnapshotsMetric,
		subsystem,
		"cache_snapshots",
		"Number of cache snapshots taken.",
		"zone",
	)
	return &cacheSnapshots{
		metric: cacheSnapshotsMetric,
	}
}

func (c *cacheSnapshots) Set(zone string, n int) {
	c.metric.WithLabelValues(zone).Set(float64(n))
}

// Catalog Members /////////////////////////////////////////////////////////////

var (
	catalogMembersOnce   sync.Once
	catalogMembersMetric *prometheus.GaugeVec
)

type catalogMembers struct {
	metric *prometheus.GaugeVec
}

func newCatalogMembers(subsystem string) *catalogMembers {
	onceGaugeVec(
		&catalogMembersOnce,
		&catalogMembersMetric,
		subsystem,
		"catalog_members",
		"Number of members in the catalog zone.",
		"catalog",
	)
	return &catalogMembers{
		metric: catalogMembersMetric,
	}
}

func (c *catalogMembers) Set(catalog string, n int) {
	c.metric.WithLabelValues(catalog).Set(float64(n))
}
