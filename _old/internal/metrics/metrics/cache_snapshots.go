package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cacheSnapshotsOnce   sync.Once
	cacheSnapshotsMetric *prometheus.GaugeVec
)

type CacheSnapshots struct {
	metric *prometheus.GaugeVec
}

func NewCacheSnapshots(subsystem string) *CacheSnapshots {
	o := prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "cache_snapshots",
		Help:      "number of cache snapshots taken",
	}
	cacheSnapshotsOnce.Do(
		func() {
			cacheSnapshotsMetric = promauto.NewGaugeVec(
				o,
				[]string{"zone"},
			)
		},
	)
	return &CacheSnapshots{
		metric: cacheSnapshotsMetric,
	}
}

func (c *CacheSnapshots) Set(zone string, n int) {
	c.metric.WithLabelValues(zone).Set(float64(n))
}
