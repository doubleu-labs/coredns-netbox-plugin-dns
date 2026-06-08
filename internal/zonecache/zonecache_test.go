package zonecache

import (
	"testing"

	"github.com/miekg/dns"
)

func mustRR(t *testing.T, s string) dns.RR {
	t.Helper()
	rr, err := dns.NewRR(s)
	if err != nil {
		t.Fatalf("dns.NewRR(%q): %v", s, err)
	}
	return rr
}

func TestCache_PutLatest(t *testing.T) {
	c := New(4)
	rr := mustRR(t, "example.com. 300 IN A 1.2.3.4")
	c.Put("example.com.", 1, []dns.RR{rr})

	got, ok := c.Latest("EXAMPLE.com")
	if !ok {
		t.Fatal("Latest: missing")
	}
	if got.Serial != 1 || len(got.RRs) != 1 {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
	// Mutating the returned slice must not affect the cache.
	got.RRs[0].Header().Ttl = 999
	again, _ := c.Latest("example.com")
	if again.RRs[0].Header().Ttl != 300 {
		t.Errorf("Latest must return a deep copy")
	}
}

func TestCache_PutSameSerialIsIdempotent(t *testing.T) {
	c := New(4)
	c.Put("example.com", 1, []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")})
	c.Put("example.com", 1, []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")})
	if got := c.Len("example.com"); got != 1 {
		t.Errorf("Len = %d, want 1", got)
	}
}

func TestCache_RingEviction(t *testing.T) {
	c := New(3)
	for s := uint32(1); s <= 5; s++ {
		c.Put("example.com", s, []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")})
	}
	if got := c.Len("example.com"); got != 3 {
		t.Fatalf("Len = %d, want 3", got)
	}
	// Oldest two (serials 1, 2) must be gone.
	if _, _, ok := c.Diff("example.com", 1); ok {
		t.Error("serial 1 should have been evicted")
	}
	if _, _, ok := c.Diff("example.com", 2); ok {
		t.Error("serial 2 should have been evicted")
	}
	if _, _, ok := c.Diff("example.com", 3); !ok {
		t.Error("serial 3 should still be present")
	}
}

func TestCache_DiffMissingZone(t *testing.T) {
	c := New(4)
	if _, _, ok := c.Diff("absent.test", 1); ok {
		t.Error("Diff on unknown zone must return ok=false")
	}
}

func TestCache_DiffReturnsLatest(t *testing.T) {
	c := New(4)
	c.Put("example.com", 1, nil)
	c.Put("example.com", 2, nil)
	from, to, ok := c.Diff("example.com", 1)
	if !ok {
		t.Fatal("ok=false")
	}
	if from.Serial != 1 || to.Serial != 2 {
		t.Errorf("from=%d to=%d, want 1,2", from.Serial, to.Serial)
	}
}

func TestCache_NewClampsMaxSize(t *testing.T) {
	c := New(0)
	c.Put("example.com", 1, nil)
	c.Put("example.com", 2, nil)
	if got := c.Len("example.com"); got != 1 {
		t.Errorf("Len = %d, want 1 (clamped)", got)
	}
}
