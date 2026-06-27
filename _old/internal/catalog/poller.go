package catalog

import (
	"slices"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/view"
)

type PollerConfig struct {
	ixfrHistory int
	interval    time.Duration
	client      *netbox.Client
	views       *view.View
	metrics     *metrics.Metrics
	logger      *log.P
	prefix      string
	tracker     *Tracker
}

func NewPollerConfig(
	ixfrHistory int,
	interval time.Duration,
	client *netbox.Client,
	views *view.View,
	metrics *metrics.Metrics,
	logger *log.P,
	prefix string,
	tracker *Tracker,
) *PollerConfig {
	return &PollerConfig{
		ixfrHistory: ixfrHistory,
		interval:    interval,
		client:      client,
		views:       views,
		metrics:     metrics,
		logger:      logger,
		prefix:      prefix,
		tracker:     tracker,
	}
}

type Poller struct {
	PollerConfig

	cache   *Cache
	catalog *Catalog

	stop chan struct{}
	done chan struct{}
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
		t := time.NewTicker(p.interval)
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

	czs, err := netbox.GetCatalogZones(p.client, p.views, p.prefix)
	if err != nil {
		p.logger.Errorf("[poller] listing catalog zones: %v", err)
		return
	}
	for cz := range slices.Values(czs) {
		s := p.tracker.NextSerial(cz.Name, zs)
	}
}
