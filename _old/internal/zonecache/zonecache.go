// Package zonecache stores a bounded history of zone snapshots so that the
// netboxdns plugin can answer IXFR (RFC 1995) requests with a real diff
// instead of always falling back to AXFR.
//
// A snapshot is a (serial, []dns.RR) pair captured by the poller after each
// observed SOA serial bump. Snapshots are kept per zone in a fixed-size ring
// buffer; the oldest entry is evicted once the ring is full. RRs in a stored
// snapshot must NOT include the SOA — Transfer() synthesises SOAs separately.
package zonecache

// Snapshot is the state of a zone at a given SOA serial.
// type Snapshot struct {
// 	Serial uint32
// 	RRs    []dns.RR
// }

// Cache is a per-zone ring buffer of snapshots. It is safe for concurrent use.
// type Cache struct {
// 	mu      sync.RWMutex
// 	maxSize int
// 	zones   map[string]*ring
// }

// type ring struct {
// 	entries []Snapshot
// 	// next index to write into. The valid window is the most recent
// 	// min(len(entries), maxSize) snapshots in insertion order.
// }

// New returns a Cache that keeps at most maxSize snapshots per zone. A
// non-positive maxSize is clamped to 1.
// func New(maxSize int) *Cache {
// 	if maxSize < 1 {
// 		maxSize = 1
// 	}
// 	return &Cache{
// 		maxSize: maxSize,
// 		zones:   make(map[string]*ring),
// 	}
// }

// Put records a new snapshot for the given zone. If the most recent snapshot
// already has the same serial, Put is a no-op (the poller calls Put on every
// tick and we want it to be idempotent when nothing changed). The zone name
// is normalised to lower case without a trailing dot.
// func (c *Cache) Put(zone string, serial uint32, rrs []dns.RR) {
// 	z := normaliseZone(zone)
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	r, ok := c.zones[z]
// 	if !ok {
// 		r = &ring{entries: make([]Snapshot, 0, c.maxSize)}
// 		c.zones[z] = r
// 	}
// 	if n := len(r.entries); n > 0 && r.entries[n-1].Serial == serial {
// 		// Refresh RRs in place — they may have been re-fetched even though
// 		// the serial did not advance, and we want the latest copy in cache.
// 		r.entries[n-1].RRs = cloneRRs(rrs)
// 		return
// 	}
// 	snap := Snapshot{Serial: serial, RRs: cloneRRs(rrs)}
// 	if len(r.entries) < c.maxSize {
// 		r.entries = append(r.entries, snap)
// 		return
// 	}
// 	// Evict oldest by shifting; maxSize is small (default 16) so the cost
// 	// is negligible compared to running an actual ring index.
// 	copy(r.entries, r.entries[1:])
// 	r.entries[c.maxSize-1] = snap
// }

// Latest returns the most recent snapshot for zone, or (Snapshot{}, false)
// when the zone has never been seen.
// func (c *Cache) Latest(zone string) (Snapshot, bool) {
// 	z := normaliseZone(zone)
// 	c.mu.RLock()
// 	defer c.mu.RUnlock()
// 	r, ok := c.zones[z]
// 	if !ok || len(r.entries) == 0 {
// 		return Snapshot{}, false
// 	}
// 	last := r.entries[len(r.entries)-1]
// 	return Snapshot{Serial: last.Serial, RRs: cloneRRs(last.RRs)}, true
// }

// Diff returns the snapshot recorded for fromSerial and the most recent
// snapshot, so the caller can compute the IXFR delta. ok is false when
// fromSerial is not present in the ring (cache miss → caller should AXFR).
//
// If fromSerial equals the latest serial, ok is true and from == to (the
// caller will detect this and return an IXFR no-op).
// func (c *Cache) Diff(zone string, fromSerial uint32) (
// 	from, to Snapshot,
// 	ok bool,
// ) {
// 	z := normaliseZone(zone)
// 	c.mu.RLock()
// 	defer c.mu.RUnlock()
// 	r, ok := c.zones[z]
// 	if !ok || len(r.entries) == 0 {
// 		return Snapshot{}, Snapshot{}, false
// 	}
// 	latest := r.entries[len(r.entries)-1]
// 	for i := range r.entries {
// 		if r.entries[i].Serial == fromSerial {
// 			return Snapshot{
// 					Serial: r.entries[i].Serial,
// 					RRs:    cloneRRs(r.entries[i].RRs),
// 				},
// 				Snapshot{Serial: latest.Serial, RRs: cloneRRs(latest.RRs)},
// 				true
// 		}
// 	}
// 	return Snapshot{}, Snapshot{}, false
// }

// Len returns the number of snapshots currently held for zone (mainly for
// tests and observability).
// func (c *Cache) Len(zone string) int {
// 	z := normaliseZone(zone)
// 	c.mu.RLock()
// 	defer c.mu.RUnlock()
// 	r, ok := c.zones[z]
// 	if !ok {
// 		return 0
// 	}
// 	return len(r.entries)
// }

// func normaliseZone(name string) string {
// 	return strings.ToLower(strings.TrimSuffix(name, "."))
// }

// func cloneRRs(in []dns.RR) []dns.RR {
// 	if in == nil {
// 		return nil
// 	}
// 	out := make([]dns.RR, len(in))
// 	for i := range in {
// 		out[i] = dns.Copy(in[i])
// 	}
// 	return out
// }
