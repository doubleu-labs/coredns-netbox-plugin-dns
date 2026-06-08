package netboxdns

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/coredns/coredns/plugin/transfer"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// fixtureZoneFull is a /zones/ envelope containing a single zone with all
// SOA fields populated. Used by the transfer tests so that buildSOA produces
// a deterministic record.
const fixtureZoneFull = `{
    "count": 1, "next": null, "previous": null,
    "results": [
        {
            "id": 1,
            "name": "example.com",
            "default_ttl": 3600,
            "nameservers": [{"name": "ns1.example.com"}, {"name": "ns2.example.com"}],
            "view": null,
            "soa_ttl": 86400,
            "soa_mname": {"name": "ns1.example.com"},
            "soa_rname": "admin.example.com",
            "soa_serial": 2026040801,
            "soa_refresh": 43200,
            "soa_retry": 7200,
            "soa_expire": 2419200,
            "soa_minimum": 3600
        }
    ]
}`

// drainTransfer drains the transfer channel into a flat slice and reports the
// number of separate batches received.
func drainTransfer(t *testing.T, ch <-chan []dns.RR) ([]dns.RR, int) {
	t.Helper()
	var all []dns.RR
	batches := 0
	for batch := range ch {
		batches++
		all = append(all, batch...)
	}
	return all, batches
}

// ----- buildSOA --------------------------------------------------------

func TestBuildSOA(t *testing.T) {
	z := &netbox.Zone{
		Name:       "example.com",
		SOATTL:     86400,
		SOAMName:   netbox.SOAMName{Name: "ns1.example.com"},
		SOARName:   "admin.example.com",
		SOASerial:  2026040801,
		SOARefresh: 43200,
		SOARetry:   7200,
		SOAExpire:  2419200,
		SOAMinimum: 3600,
	}
	soa := buildSOA(z, dns.Fqdn(z.Name))

	if soa.Hdr.Name != "example.com." {
		t.Errorf("Hdr.Name = %q, want example.com.", soa.Hdr.Name)
	}
	if soa.Hdr.Ttl != 86400 {
		t.Errorf("Hdr.Ttl = %d, want 86400", soa.Hdr.Ttl)
	}
	if soa.Ns != "ns1.example.com." {
		t.Errorf("Ns = %q, want ns1.example.com.", soa.Ns)
	}
	if soa.Mbox != "admin.example.com." {
		t.Errorf("Mbox = %q, want admin.example.com.", soa.Mbox)
	}
	if soa.Serial != 2026040801 {
		t.Errorf("Serial = %d, want 2026040801", soa.Serial)
	}
	if soa.Refresh != 43200 || soa.Retry != 7200 || soa.Expire != 2419200 || soa.Minttl != 3600 {
		t.Errorf("timing fields wrong: %+v", soa)
	}
}

// ----- Transfer: NotAuthoritative --------------------------------------

func TestTransfer_NotAuthoritative(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(`{
        "count": 0, "next": null, "previous": null, "results": []
    }`))

	ch, err := plugin.Transfer("nope.example.com.", 0)
	if !errors.Is(err, transfer.ErrNotAuthoritative) {
		t.Fatalf("err = %v, want transfer.ErrNotAuthoritative", err)
	}
	if ch != nil {
		t.Errorf("expected nil channel, got %v", ch)
	}
}

// Errors from GetZones must propagate to the caller (the transfer plugin
// turns these into SERVFAIL).
func TestTransfer_GetZonesError(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.HandleFunc("/api/plugins/netbox-dns/zones/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	_, err := plugin.Transfer("example.com.", 0)
	if err == nil {
		t.Fatal("expected error from Transfer when GetZones fails")
	}
	if errors.Is(err, transfer.ErrNotAuthoritative) {
		t.Errorf("got ErrNotAuthoritative, want a generic API error: %v", err)
	}
}

// ----- Transfer: AXFR full ----------------------------------------------

func TestTransfer_AXFR_FullZone(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, r *http.Request) {
		// Sanity-check the call: filter is by zone_id of our fixture zone.
		if got := r.URL.Query().Get("zone_id"); got != "1" {
			t.Errorf("zone_id query = %q, want 1", got)
		}
		writeJSON(w, `{
            "count": 4, "next": null, "previous": null, "results": [
                {"type": "NS",   "fqdn": "example.com.",        "absolute_value": "ns1.example.com.", "ttl": 3600, "zone": {"id": 1, "default_ttl": 3600}},
                {"type": "NS",   "fqdn": "example.com.",        "absolute_value": "ns2.example.com.", "ttl": 3600, "zone": {"id": 1, "default_ttl": 3600}},
                {"type": "A",    "fqdn": "ns1.example.com.",    "absolute_value": "10.0.0.10",        "ttl": null, "zone": {"id": 1, "default_ttl": 3600}},
                {"type": "AAAA", "fqdn": "ns1.example.com.",    "absolute_value": "2001:db8::10",     "ttl": null, "zone": {"id": 1, "default_ttl": 3600}}
            ]
        }`)
	})

	ch, err := plugin.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, batches := drainTransfer(t, ch)

	if batches < 2 {
		t.Errorf("batches = %d, want at least 2 (opening + closing SOA)", batches)
	}
	if len(rrs) < 2 {
		t.Fatalf("len = %d, want at least SOA+SOA", len(rrs))
	}

	// SOA must be first and last (RFC 5936 §2.2).
	if _, ok := rrs[0].(*dns.SOA); !ok {
		t.Errorf("first RR is %T, want *dns.SOA", rrs[0])
	}
	if _, ok := rrs[len(rrs)-1].(*dns.SOA); !ok {
		t.Errorf("last RR is %T, want *dns.SOA", rrs[len(rrs)-1])
	}

	// Body must contain all 4 records (2× NS + A + AAAA).
	body := rrs[1 : len(rrs)-1]
	if len(body) != 4 {
		t.Errorf("body len = %d, want 4 (got %v)", len(body), body)
	}

	// Sanity: at least one NS, one A, one AAAA.
	var sawNS, sawA, sawAAAA bool
	for _, rr := range body {
		switch rr.(type) {
		case *dns.NS:
			sawNS = true
		case *dns.A:
			sawA = true
		case *dns.AAAA:
			sawAAAA = true
		}
	}
	if !sawNS || !sawA || !sawAAAA {
		t.Errorf("missing record types: NS=%v A=%v AAAA=%v", sawNS, sawA, sawAAAA)
	}
}

// netbox-dns returns SOA records on /records/. Transfer must drop them and
// rely on the synthesized SOA from zone metadata to keep wire form
// deterministic.
func TestTransfer_AXFR_DropsNetboxSOA(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{
            "count": 2, "next": null, "previous": null, "results": [
                {"type": "SOA", "fqdn": "example.com.", "absolute_value": "ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600", "ttl": 86400, "zone": {"id": 1, "default_ttl": 3600}},
                {"type": "A",   "fqdn": "host.example.com.", "absolute_value": "10.0.0.20", "ttl": null, "zone": {"id": 1, "default_ttl": 3600}}
            ]
        }`)
	})

	ch, err := plugin.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)

	// We expect: SOA(synth) + A + SOA(synth). Exactly one SOA from netbox
	// must have been dropped, leaving the body with just the A record.
	if len(rrs) != 3 {
		t.Fatalf("len = %d, want 3 (synth-SOA, A, synth-SOA); got %v", len(rrs), rrs)
	}
	if _, ok := rrs[1].(*dns.A); !ok {
		t.Errorf("body[0] = %T, want *dns.A", rrs[1])
	}
	// Both SOAs must be the synthesized one (Serial from zone metadata).
	for i, rr := range []dns.RR{rrs[0], rrs[2]} {
		soa, ok := rr.(*dns.SOA)
		if !ok {
			t.Errorf("idx %d: %T, want *dns.SOA", i, rr)
			continue
		}
		if soa.Serial != 2026040801 {
			t.Errorf("idx %d: serial = %d, want synth value 2026040801", i, soa.Serial)
		}
	}
}

// ----- Transfer: IXFR no-op --------------------------------------------

func TestTransfer_IXFR_NoOp_SerialEqual(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("records endpoint should NOT be called during IXFR no-op; got %s", r.URL)
		writeJSON(w, `{"count": 0, "next": null, "previous": null, "results": []}`)
	})

	ch, err := plugin.Transfer("example.com.", 2026040801) // == current serial
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, batches := drainTransfer(t, ch)
	if batches != 1 || len(rrs) != 1 {
		t.Errorf("batches=%d rrs=%d, want 1 batch with 1 RR", batches, len(rrs))
	}
	if _, ok := rrs[0].(*dns.SOA); !ok {
		t.Errorf("rr[0] = %T, want *dns.SOA", rrs[0])
	}
}

func TestTransfer_IXFR_NoOp_SerialNewer(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("records endpoint should NOT be called; got %s", r.URL)
	})

	ch, err := plugin.Transfer("example.com.", 2026040802) // > current
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)
	if len(rrs) != 1 {
		t.Fatalf("expected 1 RR (SOA), got %d", len(rrs))
	}
}

// ----- Transfer: IXFR fallback (older serial → AXFR) --------------------

func TestTransfer_IXFR_OlderSerialFallsBackToAXFR(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{
            "count": 1, "next": null, "previous": null, "results": [
                {"type": "A", "fqdn": "host.example.com.", "absolute_value": "10.0.0.30", "ttl": null, "zone": {"id": 1, "default_ttl": 3600}}
            ]
        }`)
	})

	ch, err := plugin.Transfer("example.com.", 2026040800) // < current (2026040801)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)

	if len(rrs) != 3 {
		t.Fatalf("len = %d, want 3 (SOA + A + SOA)", len(rrs))
	}
	if _, ok := rrs[0].(*dns.SOA); !ok {
		t.Errorf("first RR not SOA: %T", rrs[0])
	}
	if _, ok := rrs[2].(*dns.SOA); !ok {
		t.Errorf("last RR not SOA: %T", rrs[2])
	}
	if a, ok := rrs[1].(*dns.A); !ok || a.A.String() != "10.0.0.30" {
		t.Errorf("middle RR not the expected A record: %#v", rrs[1])
	}
}

// ----- Transfer: batching for large zones -------------------------------

func TestTransfer_AXFR_BatchesLargeZone(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))

	// Build a fixture with 250 A records → expect at least ceil(250/100)=3
	// body batches plus opening + closing SOA = 5 batches.
	const recordCount = 250
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		var b strings.Builder
		fmt.Fprintf(&b, `{"count": %d, "next": null, "previous": null, "results": [`, recordCount)
		for i := 0; i < recordCount; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b,
				`{"type":"A","fqdn":"h%d.example.com.","absolute_value":"10.1.%d.%d","ttl":null,"zone":{"id":1,"default_ttl":3600}}`,
				i, i/256, i%256)
		}
		b.WriteString("]}")
		writeJSON(w, b.String())
	})

	ch, err := plugin.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, batches := drainTransfer(t, ch)

	// Expected: 1 (opening SOA) + ceil(250/100)=3 (body) + 1 (closing SOA) = 5
	if batches != 5 {
		t.Errorf("batches = %d, want 5", batches)
	}
	if len(rrs) != recordCount+2 {
		t.Errorf("total RRs = %d, want %d", len(rrs), recordCount+2)
	}
	if _, ok := rrs[0].(*dns.SOA); !ok {
		t.Errorf("first RR not SOA")
	}
	if _, ok := rrs[len(rrs)-1].(*dns.SOA); !ok {
		t.Errorf("last RR not SOA")
	}
}

// ----- Transfer: case-insensitive zone match ----------------------------

func TestTransfer_CaseInsensitiveMatch(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(fixtureZoneFull))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"count": 0, "next": null, "previous": null, "results": []}`)
	})

	ch, err := plugin.Transfer("EXAMPLE.COM.", 0)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	rrs, _ := drainTransfer(t, ch)
	if len(rrs) != 2 {
		t.Errorf("got %d RRs, want 2 (SOA+SOA)", len(rrs))
	}
}

// ----- Transfer: viewName plumbed through -------------------------------

func TestTransfer_HonoursViewName(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.viewName = "internal"

	var gotView string
	mux.HandleFunc("/api/plugins/netbox-dns/zones/", func(w http.ResponseWriter, r *http.Request) {
		gotView = r.URL.Query().Get("view")
		writeJSON(w, fixtureZoneFull)
	})
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"count": 0, "next": null, "previous": null, "results": []}`)
	})

	ch, err := plugin.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	_, _ = drainTransfer(t, ch)

	if gotView != "internal" {
		t.Errorf("?view query = %q, want internal", gotView)
	}
}
