package poller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/cache"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/poller"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
)

const defaultZonePollerInterval = 30 * time.Second

type ZonePoller struct {
	*poller.Poller
	activeZoneStatus []string
	client           *api.Client
	cache            *cache.Cache
	logger           *log.P
	views            *core.Views

	pollerCacheSize        *prometheus.GaugeVec
	pollerCyclesMetric     *prometheus.CounterVec
	pollerDurationMetric   prometheus.Histogram
	pollerZoneSerialMetric *prometheus.GaugeVec
}

func NewZonePoller(
	client *api.Client,
	cache *cache.Cache,
	views *core.Views,
	activeZoneStatus []string,
	logger *log.P,
	i time.Duration,
) (*ZonePoller, error) {
	if client == nil {
		return nil, errors.New("zone poller client is required")
	}
	if cache == nil {
		return nil, errors.New("zone poller cache is required")
	}
	interval := i
	if i <= 0 {
		interval = defaultZonePollerInterval
	}
	zp := &ZonePoller{
		Poller: &poller.Poller{
			ErrorMetric: newZonePollerErrorsMetric(),
			Interval:    interval,
		},
		activeZoneStatus:       activeZoneStatus,
		cache:                  cache,
		client:                 client,
		logger:                 logger,
		views:                  views,
		pollerCacheSize:        newZonePollerCacheSizeMetric(),
		pollerCyclesMetric:     newZonePollerCyclesTotalMetric(),
		pollerDurationMetric:   newZonePollerPollDurationMetric(),
		pollerZoneSerialMetric: newZonePollerZoneSerialMetric(),
	}
	zp.Poller.PollFunc = zp.poll
	return zp, nil
}

func (zp *ZonePoller) poll(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		break
	}

	zq := &api.ZoneQuery{
		ActiveZoneStatus: zp.activeZoneStatus,
		Views:            zp.views,
	}
	start := time.Now()
	defer func() {
		zp.pollerDurationMetric.Observe(time.Since(start).Seconds())
	}()
	zoneCtx, zoneCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer zoneCancel()
	zones, err := zq.GetZones(zoneCtx, zp.client)
	if err != nil {
		zp.pollerCyclesMetric.WithLabelValues("error").Inc()
		return err
	}
	zp.pollerCyclesMetric.WithLabelValues("success").Inc()

	for i := range zones {
		zone := &zones[i]
		records, recordsErr := zp.getRecordsForZone(zoneCtx, zone)
		if recordsErr != nil {
			zp.logger.Errorf(
				"%s; %v",
				core.ScopedMessage(
					"zone_poller",
					fmt.Sprintf("fetching records for zone %s", zone.Name),
				),
				recordsErr,
			)
		}
		rrs, rrsErr := records.ToRRs()
		if rrsErr != nil {
			zp.logger.Errorf(
				"%s; %v",
				core.ScopedMessage(
					"zone_poller",
					fmt.Sprintf(
						"converting records for zone %s to rrs",
						zone.Name,
					),
				),
				rrsErr,
			)
		}
		// get the first encountered NS record for the synthesized SOA record
		// MNAME
		nsIdx := slices.IndexFunc(
			rrs,
			func(rr dns.RR) bool {
				return rr.Header().Rrtype == dns.TypeNS
			},
		)
		var mname string
		if nsIdx != -1 {
			mname = rrs[nsIdx].(*dns.NS).Ns
		}
		soa := cache.SynthesizeSOA(zone, mname)
		zp.cache.Put(zone, soa, rrs)
		zp.pollerZoneSerialMetric.WithLabelValues(zone.Name).Set(
			float64(soa.Serial),
		)
		zp.pollerCacheSize.WithLabelValues(zone.Name).Set(
			float64(zp.cache.Size()),
		)
	}

	return nil
}

func (zp *ZonePoller) getRecordsForZone(
	ctx context.Context,
	zone *api.Zone,
) (api.Records, error) {
	rq := &api.RecordQuery{
		TypeExclude: []string{"SOA"},
		Zone:        zone,
	}
	recordCtx, recordCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer recordCancel()
	return rq.GetRecords(recordCtx, zp.client)
}
