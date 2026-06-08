package netboxdns

import (
	"strings"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/zonecache"
)

// startPoller spins up the IXFR snapshot poller goroutine. It is a no-op
// when ixfrHistory == 0 (IXFR delta path disabled — Transfer falls back to
// AXFR for stale serials).
//
// One immediate poll runs synchronously *after* the goroutine starts so the
// cache is warm enough to satisfy the first IXFR request without racing the
// first tick. Errors are logged and swallowed: the poller must never crash
// the plugin.
func (n *NetboxDNS) startPoller() {
	if n.ixfrHistory <= 0 {
		return
	}
	n.cache = zonecache.New(n.ixfrHistory)
	n.stopPoller = make(chan struct{})
	n.pollerDone = make(chan struct{})

	go func() {
		defer close(n.pollerDone)
		n.pollOnce()
		t := time.NewTicker(n.pollInterval)
		defer t.Stop()
		for {
			select {
			case <-n.stopPoller:
				return
			case <-t.C:
				n.pollOnce()
			}
		}
	}()
}

// stopPollerAndWait signals the poller goroutine to exit and blocks until it
// has done so. Safe to call when the poller was never started.
func (n *NetboxDNS) stopPollerAndWait() {
	if n.stopPoller == nil {
		return
	}
	close(n.stopPoller)
	<-n.pollerDone
	n.stopPoller = nil
	n.pollerDone = nil
}

// pollOnce performs one poll cycle: list zones in the configured view,
// fetch records for each, and write a snapshot if the SOA serial advanced.
// The cache's own Put is idempotent on serial collisions, so we don't need
// to track previous state separately.
//
// Catalog zones (status=parked AND name has prefix "cat.") are handled in
// the same cycle: their content is synthesised from the active member
// zones rather than fetched from /records/, and their SOA serial is
// assigned by catalogTracker so it only bumps on real membership change.
func (n *NetboxDNS) pollOnce() {
	start := time.Now()
	defer func() {
		pollDuration.Observe(time.Since(start).Seconds())
	}()
	zones, err := n.getActiveZones()
	if err != nil {
		pollCyclesTotal.WithLabelValues("error").Inc()
		logger.Errorf("poller: listing zones: %v", err)
		return
	}
	pollCyclesTotal.WithLabelValues("success").Inc()
	n.pollCatalogs(zones)
	for i := range zones {
		z := &zones[i]
		records, err := netbox.GetRecordsQuery(
			n.requestClient,
			&netbox.RecordQuery{Zone: z},
		)
		if err != nil {
			logger.Errorf("poller: fetching records for %s: %v", z.Name, err)
			continue
		}
		// Drop SOA records — Transfer synthesises them. Storing them in
		// the snapshot would only add noise to diffs.
		filtered := make([]netbox.Record, 0, len(records))
		for k := range records {
			if records[k].Type == "SOA" {
				continue
			}
			filtered = append(filtered, records[k])
		}
		rrs, err := recordsToRR(filtered)
		if err != nil {
			logger.Errorf("poller: converting records for %s: %v", z.Name, err)
			continue
		}
		n.cache.Put(z.Name, z.SOASerial, rrs)
		zoneSerial.WithLabelValues(z.Name).Set(float64(z.SOASerial))
		cacheSnapshots.WithLabelValues(z.Name).Set(float64(n.cache.Len(z.Name)))
	}
}

// pollCatalogs is the catalog-zone half of pollOnce. It fetches the
// parked+cat.* zones in the configured view, synthesises their RFC 9432
// content from the active member zones, and writes one snapshot per
// catalog into the cache. Errors are logged and swallowed so a broken
// catalog never knocks out the normal serving path.
func (n *NetboxDNS) pollCatalogs(activeZones []netbox.Zone) {
	catalogs, err := n.getCatalogZones()
	if err != nil {
		logger.Errorf("poller: listing catalog zones: %v", err)
		return
	}
	for i := range catalogs {
		c := &catalogs[i]
		serial := n.catalogTracker.NextSerial(c.Name, activeZones)
		rrs := buildCatalog(c, activeZones)
		n.cache.Put(c.Name, serial, rrs)
		zoneSerial.WithLabelValues(c.Name).Set(float64(serial))
		cacheSnapshots.WithLabelValues(c.Name).Set(float64(n.cache.Len(c.Name)))
		catalogMembers.WithLabelValues(c.Name).Set(float64(catalogMemberCount(c, activeZones)))
	}
}

// catalogMemberCount returns how many active zones from the same view are
// published in the given catalog. Uses the same name-based exclusion as
// buildCatalog so the gauge exactly matches the synthesised PTR count.
func catalogMemberCount(catalog *netbox.Zone, active []netbox.Zone) int {
	cLower := strings.ToLower(strings.TrimSuffix(catalog.Name, "."))
	n := 0
	for i := range active {
		if strings.EqualFold(active[i].Name, cLower) {
			continue
		}
		n++
	}
	return n
}
