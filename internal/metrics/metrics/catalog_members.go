package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	catalogMembersOnce   sync.Once
	catalogMembersMetric *prometheus.GaugeVec
)

type CatalogMembers struct {
	metric *prometheus.GaugeVec
}

func NewCatalogMembers(subsystem string) *CatalogMembers {
	o := prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "catalog_members",
		Help:      "number of members in the catalog zone",
	}
	catalogMembersOnce.Do(
		func() {
			catalogMembersMetric = promauto.NewGaugeVec(
				o,
				[]string{"catalog"},
			)
		},
	)
	return &CatalogMembers{
		metric: catalogMembersMetric,
	}
}

func (c *CatalogMembers) Set(catalog string, n int) {
	c.metric.WithLabelValues(catalog).Set(float64(n))
}
