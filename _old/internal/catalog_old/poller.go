package catalog_old

import (
	"slices"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/view"
)

type Poller struct {
	ixfrHistory int
	cache       *Cache
	tracker     *Tracker

	logger  *log.P
	metrics *metrics.Metrics
	client  *netbox.Client
	views   *view.View
	prefix  *string

	stop chan struct{}
	done chan struct{}
}

func NewPoller(
	l *log.P,
	m *metrics.Metrics,
	c *netbox.Client,
	v *view.View,
	p *string,
) *Poller {
	return &Poller{
		logger:  l,
		metrics: m,
		client:  c,
		views:   v,
		prefix:  p,
	}
}

func (p *Poller) Start() {
	if p.ixfrHistory <= 0 {
		return
	}
	p.cache = NewCache(p.ixfrHistory)
	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		p.poll()
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-t.C:
				p.poll()
			}
		}
	}()
}

func (p *Poller) StopAndWait() {
	if p.stop == nil {
		return
	}
	stop := p.stop
	done := p.done
	p.stop = nil
	p.done = nil
	close(stop)
	<-done
}

func (p *Poller) poll() {
	start := time.Now()
	defer func() {
		p.metrics.PollDuration.Observe(time.Since(start).Seconds())
	}()
	zs, err := netbox.GetZones(p.client, p.views)
	if err != nil {
		p.metrics.PollCyclesTotal.Inc("error")
		p.logger.Errorf("[poller] listing zones: %v", err)
		return
	}
	p.metrics.PollCyclesTotal.Inc("success")

	czs, err := netbox.GetCatalogZones(p.client, p.views, *p.prefix)
	if err != nil {
		p.logger.Errorf("[poller] listing catalog zones: %v", err)
		return
	}
	for cz := range slices.Values(czs) {
		s := p.tracker.NextSerial(cz.Name, zs)
		rrs := buildCatalog(&cz, zs)
		p.cache.Put(cz.Name, s, rrs)
		p.metrics.ZoneSerial.Set(cz.Name, s)
		p.metrics.CacheSnapshots.Set(cz.Name, p.cache.Len(cz.Name))
		p.metrics.CatalogMembers.Set(cz.Name, catalogMemberCount(&cz, zs))
	}

	for z := range slices.Values(zs) {
		var zsErr error
		rq := netbox.RecordQuery{
			TypeExclude: []string{"SOA"},
			Zone:        &z,
		}
		recs, zsErr := rq.GetRecords(p.client)
		if zsErr != nil {
			p.logger.Errorf(
				"[poller] fetching records for %s: %v",
				z.Name,
				zsErr,
			)
			continue
		}
		rrs, zsErr := recs.ToRRs()
		if zsErr != nil {
			p.logger.Errorf(
				"[poller] converting records for %s: %v",
				z.Name,
				zsErr,
			)
			continue
		}
		p.cache.Put(z.Name, z.SOASerial, rrs)
		p.metrics.ZoneSerial.Set(z.Name, z.SOASerial)
		p.metrics.CacheSnapshots.Set(z.Name, p.cache.Len(z.Name))
	}
}
