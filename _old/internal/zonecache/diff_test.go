package zonecache

import (
	"sort"
	"testing"

	"github.com/miekg/dns"
)

func rrStrings(rrs []dns.RR) []string {
	out := make([]string, len(rrs))
	for i, rr := range rrs {
		out[i] = rr.String()
	}
	sort.Strings(out)
	return out
}

func TestDiff_NoChange(t *testing.T) {
	old := []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")}
	new := []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")}
	removed, added := Diff(old, new)
	if len(removed) != 0 || len(added) != 0 {
		t.Errorf("expected empty diff, got removed=%v added=%v", removed, added)
	}
}

func TestDiff_AddOnly(t *testing.T) {
	old := []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")}
	new := []dns.RR{
		mustRR(t, "example.com. 300 IN A 1.2.3.4"),
		mustRR(t, "www.example.com. 300 IN A 5.6.7.8"),
	}
	removed, added := Diff(old, new)
	if len(removed) != 0 {
		t.Errorf("removed: %v", removed)
	}
	if len(added) != 1 || added[0].Header().Name != "www.example.com." {
		t.Errorf("added: %v", added)
	}
}

func TestDiff_RemoveOnly(t *testing.T) {
	old := []dns.RR{
		mustRR(t, "example.com. 300 IN A 1.2.3.4"),
		mustRR(t, "www.example.com. 300 IN A 5.6.7.8"),
	}
	new := []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")}
	removed, added := Diff(old, new)
	if len(added) != 0 {
		t.Errorf("added: %v", added)
	}
	if len(removed) != 1 || removed[0].Header().Name != "www.example.com." {
		t.Errorf("removed: %v", removed)
	}
}

func TestDiff_TTLChange(t *testing.T) {
	old := []dns.RR{mustRR(t, "example.com. 300 IN A 1.2.3.4")}
	new := []dns.RR{mustRR(t, "example.com. 600 IN A 1.2.3.4")}
	removed, added := Diff(old, new)
	if len(removed) != 1 || len(added) != 1 {
		t.Fatalf("want 1 removed + 1 added, got %d/%d", len(removed), len(added))
	}
	if removed[0].Header().Ttl != 300 || added[0].Header().Ttl != 600 {
		t.Errorf("ttl mismatch: removed=%d added=%d",
			removed[0].Header().Ttl, added[0].Header().Ttl)
	}
}

func TestDiff_IgnoresSOA(t *testing.T) {
	soa := mustRR(t, "example.com. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600")
	old := []dns.RR{soa, mustRR(t, "example.com. 300 IN A 1.2.3.4")}
	new := []dns.RR{soa}
	removed, added := Diff(old, new)
	if len(added) != 0 {
		t.Errorf("added should be empty (SOA filtered): %v", added)
	}
	if len(removed) != 1 {
		t.Fatalf("want 1 removed, got %d", len(removed))
	}
	if removed[0].Header().Rrtype != dns.TypeA {
		t.Errorf("removed should be the A record, got %v", removed[0])
	}
}

func TestDiff_CaseInsensitiveOwner(t *testing.T) {
	old := []dns.RR{mustRR(t, "WWW.example.com. 300 IN A 1.2.3.4")}
	new := []dns.RR{mustRR(t, "www.example.com. 300 IN A 1.2.3.4")}
	removed, added := Diff(old, new)
	// dns.NewRR canonicalises the owner, so this is mostly a sanity check
	// that our key path doesn't introduce a regression.
	if len(removed) != 0 || len(added) != 0 {
		t.Errorf("case difference must not produce a diff: removed=%v added=%v",
			rrStrings(removed), rrStrings(added))
	}
}
