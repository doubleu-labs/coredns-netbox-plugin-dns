package cache

import (
	"maps"
	"slices"
	"sync"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

const transferBatchSize = 100

type zoneDelta struct {
	added      []dns.RR
	deleted    []dns.RR
	fromSerial uint32
	toSerial   uint32
}

type zoneCache struct {
	currentSerial uint32
	currentZone   []dns.RR
	deltas        []zoneDelta
	soa           *dns.SOA
}

type Cache struct {
	mu sync.RWMutex

	maxHistory int
	zones      map[string]*zoneCache
}

func NewCache(maxHistory int) *Cache {
	return &Cache{
		maxHistory: maxHistory,
		zones:      make(map[string]*zoneCache),
	}
}

func (c *Cache) Transfer(zone string, serial uint32) (
	<-chan []dns.RR,
	error,
) {
	normalizedZone := normalizeZoneName(zone)

	c.mu.RLock()
	defer c.mu.RUnlock()

	zCache, exists := c.zones[normalizedZone]
	if !exists {
		return nil, ErrZoneNotFound
	}

	snapshot := snapshotZoneCache(zCache)
	if snapshot.soa == nil {
		return nil, ErrZoneHasNoSOA
	}

	ch := make(chan []dns.RR)

	// axfr request
	if serial == 0 {
		go streamAXFR(ch, snapshot)
		return ch, nil
	}

	// client up to date
	if !serialBefore(serial, snapshot.currentSerial) {
		go streamCurrentSOA(ch, snapshot.soa)
		return ch, nil
	}

	// ixfr request
	result := ixfrDeltasFrom(snapshot, serial)
	if result.fallbackAXFR {
		go streamAXFR(ch, snapshot)
		return ch, nil
	}

	go streamIXFR(ch, snapshot.soa, result.deltas)

	return ch, nil
}

func (c *Cache) GetZoneNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return slices.Sorted(maps.Keys(c.zones))
}

func (c *Cache) Put(zone *api.Zone, soa *dns.SOA, rrs []dns.RR) {
	normalizedZone := normalizeZoneName(zone.Name)

	c.mu.Lock()
	defer c.mu.Unlock()

	zCache, exists := c.zones[normalizedZone]
	if !exists {
		c.zones[normalizedZone] = &zoneCache{
			currentSerial: zone.SOASerial,
			currentZone:   rrs,
			soa:           soa,
		}
		zCache = c.zones[normalizedZone]
	}

	if zCache.currentSerial == zone.SOASerial {
		zCache.currentZone = rrs
		zCache.soa = soa
		return
	}

	if serialBefore(zone.SOASerial, zCache.currentSerial) {
		return
	}

	delta := diffZoneRRs(
		zCache.currentZone,
		rrs,
		zCache.currentSerial,
		zone.SOASerial,
	)
	zCache.deltas = append(zCache.deltas, delta)
	if c.maxHistory >= 0 && len(zCache.deltas) > c.maxHistory {
		zCache.deltas = zCache.deltas[len(zCache.deltas)-c.maxHistory:]
	}

	zCache.currentSerial = zone.SOASerial
	zCache.currentZone = rrs
	zCache.soa = soa
}

func (c *Cache) Size() int {
	var out int
	c.mu.RLock()
	defer c.mu.RUnlock()
	for z := range maps.Values(c.zones) {
		out += len(z.deltas) + 1
	}
	return out
}

func diffZoneRRs(from, to []dns.RR, fromSerial, toSerial uint32) zoneDelta {
	fromRRs := make(map[string]dns.RR, len(from))
	for rr := range slices.Values(from) {
		if rr == nil {
			continue
		}
		fromRRs[rr.String()] = rr
	}

	toRRs := make(map[string]dns.RR, len(to))
	for rr := range slices.Values(to) {
		if rr == nil {
			continue
		}
		toRRs[rr.String()] = rr
	}

	delta := zoneDelta{
		fromSerial: fromSerial,
		toSerial:   toSerial,
	}

	for key, rr := range fromRRs {
		if _, exists := toRRs[key]; !exists {
			delta.deleted = append(delta.deleted, dns.Copy(rr))
		}
	}

	for key, rr := range toRRs {
		if _, exists := fromRRs[key]; !exists {
			delta.added = append(delta.added, dns.Copy(rr))
		}
	}

	return delta
}
