package catalog_old

import (
	"slices"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

type Snapshot struct {
	Serial uint32
	RRs    []dns.RR
}

type Cache struct {
	mu      sync.RWMutex
	maxSize int
	zones   map[string]*ring
}

type ring struct {
	entries []Snapshot
}

func NewCache(maxSize int) *Cache {
	if maxSize < 1 {
		maxSize = 1
	}
	return &Cache{
		maxSize: maxSize,
		zones:   make(map[string]*ring),
	}
}

func (c *Cache) Put(z string, s uint32, rrs []dns.RR) {
	z = strings.ToLower(strings.TrimSuffix(z, "."))
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.zones[z]
	if !ok {
		r = &ring{
			entries: make([]Snapshot, 0, c.maxSize),
		}
		c.zones[z] = r
	}
	if n := len(r.entries); n > 0 && r.entries[n-1].Serial == s {
		r.entries[n-1].RRs = cloneRRs(rrs)
		return
	}
	snap := Snapshot{Serial: s, RRs: cloneRRs(rrs)}
	if len(r.entries) < c.maxSize {
		r.entries = append(r.entries, snap)
		return
	}
	copy(r.entries, r.entries[1:])
	r.entries[c.maxSize-1] = snap
}

func (c *Cache) Len(z string) int {
	z = strings.ToLower(strings.TrimSuffix(z, "."))
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, ok := c.zones[z]
	if !ok {
		return 0
	}
	return len(r.entries)
}

func (c *Cache) Latest(z string) (Snapshot, bool) {
	z = strings.ToLower(strings.TrimSuffix(z, "."))
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, ok := c.zones[z]
	if !ok || len(r.entries) == 0 {
		return Snapshot{}, false
	}
	l := r.entries[len(r.entries)-1]
	s := Snapshot{
		Serial: l.Serial,
		RRs:    cloneRRs(l.RRs),
	}
	return s, true
}

func (c *Cache) Diff(z string, s uint32) (from, to Snapshot, ok bool) {
	from = Snapshot{}
	to = Snapshot{}
	z = strings.ToLower(strings.TrimSuffix(z, "."))
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, rok := c.zones[z]
	if !rok || len(r.entries) == 0 {
		ok = false
		return
	}
	latest := r.entries[len(r.entries)-1]
	for e := range slices.Values(r.entries) {
		if e.Serial == s {
			from.Serial = e.Serial
			from.RRs = cloneRRs(e.RRs)
			to.Serial = latest.Serial
			to.RRs = cloneRRs(latest.RRs)
			ok = true
			return
		}
	}
	return
}

func cloneRRs(in []dns.RR) []dns.RR {
	if in == nil {
		return nil
	}
	out := make([]dns.RR, len(in))
	for i := range in {
		out[i] = dns.Copy(in[i])
	}
	return out
}
