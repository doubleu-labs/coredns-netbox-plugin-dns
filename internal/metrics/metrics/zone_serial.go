package metrics

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	zoneSerialOnce   sync.Once
	zoneSerialMetric *prometheus.GaugeVec
)

type ZoneSerial struct {
	metric *prometheus.GaugeVec
}

func NewZoneSerial(subsystem string) *ZoneSerial {
	o := prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: subsystem,
		Name:      "zone_serial",
		Help:      "current serial number of the zone",
	}
	zoneSerialOnce.Do(
		func() {
			zoneSerialMetric = promauto.NewGaugeVec(
				o,
				[]string{"zone"},
			)
		},
	)
	return &ZoneSerial{
		metric: zoneSerialMetric,
	}
}

func (z *ZoneSerial) Set(zone string, serial uint32) {
	z.metric.WithLabelValues(zone).Set(float64(serial))
}
