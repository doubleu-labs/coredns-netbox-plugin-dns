package metrics

import (
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics/metrics"
)

type Metrics struct {
	RequestsTotal   *metrics.RequestsTotal
	RequestDuration *metrics.RequestsDuration

	NetboxRequestsTotal   *metrics.NetboxRequestsTotal
	NetboxRequestDuration *metrics.NetboxRequestDuration

	TransfersTotal *metrics.TransfersTotal

	PollCyclesTotal *metrics.PollCyclesTotal
	PollDuration    metrics.PollDuration

	ZoneSerial     *metrics.ZoneSerial
	CacheSnapshots *metrics.CacheSnapshots
	CatalogMembers *metrics.CatalogMembers
}

func NewMetrics(subsystem string) *Metrics {
	return &Metrics{
		RequestsTotal:         metrics.NewRequestsTotal(subsystem),
		RequestDuration:       metrics.NewRequestsDuration(subsystem),
		NetboxRequestsTotal:   metrics.NewNetboxRequestsTotal(subsystem),
		NetboxRequestDuration: metrics.NewNetboxRequestDuration(subsystem),
		TransfersTotal:        metrics.NewTransfersTotal(subsystem),
		PollCyclesTotal:       metrics.NewPollCyclesTotal(subsystem),
		PollDuration:          metrics.NewPollDuration(subsystem),
		ZoneSerial:            metrics.NewZoneSerial(subsystem),
		CacheSnapshots:        metrics.NewCacheSnapshots(subsystem),
		CatalogMembers:        metrics.NewCatalogMembers(subsystem),
	}
}
