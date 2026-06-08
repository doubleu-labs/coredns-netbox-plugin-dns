package netboxdns

import (
	"net/http"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/zonecache"
	"github.com/miekg/dns"
)

// primeCache puts a snapshot into plugin.cache for example.com. Helper used
// by the IXFR delta tests below.
func primeCache(t *testing.T, plugin *NetboxDNS, serial uint32, lines ...string) {
	t.Helper()
	rrs := make([]dns.RR, 0, len(lines))
	for _, line := range lines {
		rr, err := dns.NewRR(line)
		if err != nil {
			t.Fatalf("dns.NewRR(%q): %v", line, err)
		}
		rrs = append(rrs, rr)
	}
	plugin.cache.Put("example.com", serial, rrs)
}

// TestTransfer_IXFR_DeltaFromCache verifies the RFC 1995 §4 wire layout:
//
//	newSOA
//	oldSOA  removed...  newSOA  added...
//	newSOA
func TestTransfer_IXFR_DeltaFromCache(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.cache = zonecache.New(8)

	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))

	// Old serial known to the cache; new serial == fixtureZoneFull's
	// soa_serial (2026040801).
	primeCache(t, plugin, 2026040800,
		"a.example.com. 300 IN A 1.1.1.1",
		"keep.example.com. 300 IN A 9.9.9.9",
	)
	primeCache(t, plugin, 2026040801,
		"keep.example.com. 300 IN A 9.9.9.9",
		"b.example.com. 300 IN A 2.2.2.2",
	)

	ch, err := plugin.Transfer("example.com.", 2026040800)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)

	// Layout: newSOA, oldSOA, removed(1), newSOA, added(1), newSOA
	if len(rrs) != 6 {
		t.Fatalf("len(rrs)=%d, want 6; got: %v", len(rrs), rrs)
	}
	soaNew, ok := rrs[0].(*dns.SOA)
	if !ok || soaNew.Serial != 2026040801 {
		t.Errorf("rrs[0] should be new SOA serial 2026040801, got %v", rrs[0])
	}
	soaOld, ok := rrs[1].(*dns.SOA)
	if !ok || soaOld.Serial != 2026040800 {
		t.Errorf("rrs[1] should be old SOA serial 2026040800, got %v", rrs[1])
	}
	if rrs[2].Header().Name != "a.example.com." {
		t.Errorf("rrs[2] (removed) = %v, want a.example.com.", rrs[2])
	}
	soaSep, ok := rrs[3].(*dns.SOA)
	if !ok || soaSep.Serial != 2026040801 {
		t.Errorf("rrs[3] should be separator new SOA, got %v", rrs[3])
	}
	if rrs[4].Header().Name != "b.example.com." {
		t.Errorf("rrs[4] (added) = %v, want b.example.com.", rrs[4])
	}
	soaEnd, ok := rrs[5].(*dns.SOA)
	if !ok || soaEnd.Serial != 2026040801 {
		t.Errorf("rrs[5] should be closing new SOA, got %v", rrs[5])
	}
}

// When the requested fromSerial is not in the cache (cache miss / cold
// start) Transfer must fall back to AXFR — i.e. it should NOT return an
// IXFR-shaped reply but a full zone with SOA first/last.
func TestTransfer_IXFR_CacheMissFallsBackToAXFR(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.cache = zonecache.New(8)

	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureRecordsForPoller)
	})

	// Cache holds a different fromSerial than what the client asks for.
	primeCache(t, plugin, 2026040801, "a.example.com. 300 IN A 1.1.1.1")

	ch, err := plugin.Transfer("example.com.", 19990101)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)

	// AXFR shape: SOA, body..., SOA. fixtureRecordsForPoller has 2 RRs.
	if len(rrs) != 4 {
		t.Fatalf("len(rrs)=%d, want 4 (SOA + 2 + SOA); got: %v", len(rrs), rrs)
	}
	if _, ok := rrs[0].(*dns.SOA); !ok {
		t.Errorf("rrs[0] should be SOA")
	}
	if _, ok := rrs[len(rrs)-1].(*dns.SOA); !ok {
		t.Errorf("last RR should be SOA")
	}
}

// Cache hit but the cached "to" serial does not match the live zone serial
// (poller hasn't caught up yet) → AXFR fallback.
func TestTransfer_IXFR_StaleCacheFallsBackToAXFR(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.cache = zonecache.New(8)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureRecordsForPoller)
	})

	// Cache "to" serial (2026040000) is older than live zone's 2026040801.
	primeCache(t, plugin, 2026039999, "a.example.com. 300 IN A 1.1.1.1")
	primeCache(t, plugin, 2026040000, "b.example.com. 300 IN A 2.2.2.2")

	ch, err := plugin.Transfer("example.com.", 2026039999)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)
	if len(rrs) != 4 {
		t.Errorf("expected AXFR fallback (4 RRs), got %d: %v", len(rrs), rrs)
	}
}
