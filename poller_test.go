package netboxdns

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/zonecache"
)

// pollerRecordsHandler emits an absolute_value-bearing /records/ envelope
// for a single zone_id. fixtureRecordsForPoller below uses TXT records to
// keep the rdata trivial and unambiguous.
const fixtureRecordsForPoller = `{
    "count": 2, "next": null, "previous": null,
    "results": [
        {"id": 1, "fqdn": "a.example.com.", "name": "a", "type": "TXT",
         "value": "\"v1\"", "absolute_value": "\"v1\"", "ttl": 300,
         "zone": {"id": 1}},
        {"id": 2, "fqdn": "b.example.com.", "name": "b", "type": "TXT",
         "value": "\"v2\"", "absolute_value": "\"v2\"", "ttl": 300,
         "zone": {"id": 1}}
    ]
}`

func TestPollOnce_PopulatesCache(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.cache = zonecache.New(4)

	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureRecordsForPoller)
	})

	plugin.pollOnce()

	snap, ok := plugin.cache.Latest("example.com")
	if !ok {
		t.Fatal("cache miss after pollOnce")
	}
	if snap.Serial != 2026040801 {
		t.Errorf("Serial = %d, want 2026040801", snap.Serial)
	}
	if len(snap.RRs) != 2 {
		t.Errorf("len(RRs) = %d, want 2", len(snap.RRs))
	}
	for _, rr := range snap.RRs {
		if rr.Header().Rrtype == 6 { // dns.TypeSOA
			t.Errorf("snapshot must not contain SOA: %v", rr)
		}
	}
}

func TestPollOnce_NoSerialBumpIsIdempotent(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.cache = zonecache.New(4)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureRecordsForPoller)
	})

	plugin.pollOnce()
	plugin.pollOnce()
	plugin.pollOnce()

	if got := plugin.cache.Len("example.com"); got != 1 {
		t.Errorf("Len = %d, want 1 (no serial bump)", got)
	}
}

// pollOnce must not panic or stop on a per-zone error: the records handler
// returns 500, but the cache should remain empty (not crash) and no goroutine
// should leak.
func TestPollOnce_RecordsErrorIsSwallowed(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.cache = zonecache.New(4)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))

	var hits int32
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Error(w, "boom", 500)
	})

	plugin.pollOnce() // must not panic

	if got := plugin.cache.Len("example.com"); got != 0 {
		t.Errorf("Len = %d, want 0 on records error", got)
	}
	if atomic.LoadInt32(&hits) == 0 {
		t.Error("records handler was never called")
	}
}

// TestPollerLifecycle exercises startPoller + stopPollerAndWait. We use a
// very short poll interval so the goroutine ticks at least once before we
// stop it, and assert that stop returns promptly without leaking.
func TestPollerLifecycle(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.ixfrHistory = 4
	plugin.pollInterval = 10 * time.Millisecond

	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureRecordsForPoller)
	})

	plugin.startPoller()
	// Sync poll already ran inside the goroutine; give the ticker a chance.
	time.Sleep(30 * time.Millisecond)
	plugin.stopPollerAndWait()

	if got := plugin.cache.Len("example.com"); got == 0 {
		t.Error("expected at least one snapshot after lifecycle run")
	}
	// Idempotent stop must not panic.
	plugin.stopPollerAndWait()
}

// startPoller is a no-op when ixfrHistory == 0.
func TestStartPoller_DisabledByZeroHistory(t *testing.T) {
	_, plugin := newMockPlugin(t)
	plugin.ixfrHistory = 0
	plugin.startPoller()
	if plugin.cache != nil {
		t.Error("cache must remain nil when ixfr_history == 0")
	}
	if plugin.stopPoller != nil {
		t.Error("stopPoller must remain nil when ixfr_history == 0")
	}
}
