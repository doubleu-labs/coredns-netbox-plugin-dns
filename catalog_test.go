package netboxdns

import (
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// fixtureCatalogZone is a single parked zone whose name has the cat. prefix.
// SOA fields are populated so buildSOAWithSerial returns a valid record.
const fixtureCatalogZone = `{
    "count": 1, "next": null, "previous": null,
    "results": [
        {
            "id": 99,
            "name": "cat.example.com",
            "status": "parked",
            "default_ttl": 3600,
            "nameservers": [{"name": "ns1.example.com"}, {"name": "ns2.example.com"}],
            "view": null,
            "soa_ttl": 3600,
            "soa_mname": {"name": "ns1.example.com"},
            "soa_rname": "catalog-admin.example.com",
            "soa_serial": 1,
            "soa_refresh": 3600,
            "soa_retry": 600,
            "soa_expire": 604800,
            "soa_minimum": 3600
        }
    ]
}`

// fixtureActiveZonesForCatalog returns two active member zones with stable
// IDs (3, 7) so the synthesised PTR owner names are deterministic.
const fixtureActiveZonesForCatalog = `{
    "count": 2, "next": null, "previous": null,
    "results": [
        {"id": 3, "name": "example.com", "status": "active",
         "default_ttl": 3600, "nameservers": [{"name": "ns1.example.com"}],
         "soa_ttl": 86400, "soa_mname": {"name": "ns1.example.com"},
         "soa_rname": "admin.example.com", "soa_serial": 100,
         "soa_refresh": 43200, "soa_retry": 7200,
         "soa_expire": 2419200, "soa_minimum": 3600},
        {"id": 7, "name": "internal.example.com", "status": "active",
         "default_ttl": 3600, "nameservers": [{"name": "ns1.example.com"}],
         "soa_ttl": 86400, "soa_mname": {"name": "ns1.example.com"},
         "soa_rname": "admin.example.com", "soa_serial": 200,
         "soa_refresh": 43200, "soa_retry": 7200,
         "soa_expire": 2419200, "soa_minimum": 3600}
    ]
}`

// ----- buildCatalog ----------------------------------------------------

func TestBuildCatalog_Structure(t *testing.T) {
	catalog := &netbox.Zone{
		ID:     99,
		Name:   "cat.example.com",
		Status: "parked",
		SOATTL: 3600,
		Nameservers: []netbox.Nameserver{
			{Name: "ns1.example.com"},
			{Name: "ns2.example.com"},
		},
	}
	members := []netbox.Zone{
		{ID: 3, Name: "example.com", Status: "active"},
		{ID: 7, Name: "internal.example.com", Status: "active"},
	}

	rrs := buildCatalog(catalog, members)

	// Expected layout: 2 NS + 1 version TXT + 2 PTR = 5 records.
	if len(rrs) != 5 {
		t.Fatalf("len(rrs) = %d, want 5; got: %v", len(rrs), rrs)
	}

	var nsCount, txtCount, ptrCount int
	var versionTXT *dns.TXT
	ptrs := map[string]string{}
	for _, rr := range rrs {
		switch v := rr.(type) {
		case *dns.NS:
			nsCount++
			if v.Hdr.Name != "cat.example.com." {
				t.Errorf("NS owner = %q, want cat.example.com.", v.Hdr.Name)
			}
		case *dns.TXT:
			txtCount++
			versionTXT = v
		case *dns.PTR:
			ptrCount++
			ptrs[v.Hdr.Name] = v.Ptr
		}
	}
	if nsCount != 2 {
		t.Errorf("NS count = %d, want 2", nsCount)
	}
	if txtCount != 1 {
		t.Errorf("TXT count = %d, want 1", txtCount)
	}
	if ptrCount != 2 {
		t.Errorf("PTR count = %d, want 2", ptrCount)
	}
	if versionTXT == nil || versionTXT.Hdr.Name != "version.cat.example.com." ||
		len(versionTXT.Txt) != 1 || versionTXT.Txt[0] != "2" {
		t.Errorf("version TXT wrong: %+v", versionTXT)
	}
	if ptrs["id-3.zones.cat.example.com."] != "example.com." {
		t.Errorf("PTR for id-3 missing or wrong: %v", ptrs)
	}
	if ptrs["id-7.zones.cat.example.com."] != "internal.example.com." {
		t.Errorf("PTR for id-7 missing or wrong: %v", ptrs)
	}
}

func TestBuildCatalog_ExcludesItself(t *testing.T) {
	catalog := &netbox.Zone{
		ID:     99,
		Name:   "cat.example.com",
		Status: "parked",
		SOATTL: 3600,
	}
	// Defensive: even if the active list bizarrely contains the catalog
	// itself, buildCatalog must skip it (no self-reference PTR).
	members := []netbox.Zone{
		{ID: 99, Name: "cat.example.com"},
		{ID: 3, Name: "example.com"},
	}
	rrs := buildCatalog(catalog, members)
	for _, rr := range rrs {
		if ptr, ok := rr.(*dns.PTR); ok && ptr.Ptr == "cat.example.com." {
			t.Errorf("catalog must not list itself as a member: %v", ptr)
		}
	}
}

// ----- catalogTracker ---------------------------------------------------

func TestCatalogTracker_BumpsOnMembershipChange(t *testing.T) {
	tr := newCatalogTracker()
	m1 := []netbox.Zone{{ID: 1}, {ID: 2}}
	m2 := []netbox.Zone{{ID: 1}, {ID: 2}, {ID: 3}}

	s1 := tr.NextSerial("cat.example.com", m1)
	s1again := tr.NextSerial("cat.example.com", m1)
	if s1 != s1again {
		t.Errorf(
			"serial must be stable for unchanged membership: %d vs %d",
			s1,
			s1again,
		)
	}
	s2 := tr.NextSerial("cat.example.com", m2)
	if s2 != s1+1 {
		t.Errorf("serial must bump by 1 on membership change: %d -> %d", s1, s2)
	}
}

func TestCatalogTracker_PerCatalogIndependent(t *testing.T) {
	tr := newCatalogTracker()
	a := tr.NextSerial("cat.a", []netbox.Zone{{ID: 1}})
	b := tr.NextSerial("cat.b", []netbox.Zone{{ID: 1}})
	// Both initialised from time.Now().Unix(); they may equal each other
	// but each should advance independently when its OWN membership shifts.
	tr.NextSerial("cat.a", []netbox.Zone{{ID: 1}, {ID: 2}})
	if got := tr.NextSerial("cat.b", []netbox.Zone{{ID: 1}}); got != b {
		t.Errorf("cat.b serial moved unexpectedly: %d vs %d", got, b)
	}
	if got := tr.NextSerial(
		"cat.a",
		[]netbox.Zone{{ID: 1}, {ID: 2}},
	); got != a+1 {
		t.Errorf("cat.a final serial = %d, want %d", got, a+1)
	}
}

// ----- transferCatalog (cold start: no cache yet) ----------------------

// func TestTransferCatalog_ColdStartAXFR(t *testing.T) {
// 	mux, plugin := newMockPlugin(t)
// 	plugin.cache = zonecache.New(8) // enabled but empty
//
// 	mux.HandleFunc(
// 		"/api/plugins/netbox-dns/zones/", catalogZonesHandler(
// 			fixtureActiveZonesForCatalog, fixtureCatalogZone,
// 		),
// 	)
//
// 	ch, err := plugin.Transfer("cat.example.com.", 0)
// 	if err != nil {
// 		t.Fatalf("Transfer: %v", err)
// 	}
// 	rrs, _ := drainTransfer(t, ch)
//
// 	// AXFR shape: SOA, body..., SOA. Body = 2 NS + 1 TXT + 2 PTR = 5.
// 	if len(rrs) != 7 {
// 		t.Fatalf("len(rrs) = %d, want 7 (SOA + 5 + SOA); got %v", len(rrs), rrs)
// 	}
// 	soaOpen, ok := rrs[0].(*dns.SOA)
// 	if !ok || soaOpen.Hdr.Name != "cat.example.com." {
// 		t.Errorf("rrs[0] should be cat.example.com SOA, got %v", rrs[0])
// 	}
// 	if _, ok := rrs[len(rrs)-1].(*dns.SOA); !ok {
// 		t.Errorf("last RR should be SOA")
// 	}
// 	// Catalog SOA serial must come from catalogTracker, NOT from
// 	// nbZone.SOASerial (which is 1 in fixtureCatalogZone).
// 	if soaOpen.Serial == 1 {
// 		t.Errorf("catalog SOA serial must come from tracker, got 1 (nbZone value)")
// 	}
// }

// IXFR no-op against the current catalog serial: a single SOA, channel
// closed, just like for member zones.
// func TestTransferCatalog_IXFRNoOp(t *testing.T) {
// 	mux, plugin := newMockPlugin(t)
// 	plugin.cache = zonecache.New(8)
//
// 	mux.HandleFunc(
// 		"/api/plugins/netbox-dns/zones/", catalogZonesHandler(
// 			fixtureActiveZonesForCatalog, fixtureCatalogZone,
// 		),
// 	)
//
// 	// Cold-start AXFR populates the cache so we know the current serial.
// 	ch, err := plugin.Transfer("cat.example.com.", 0)
// 	if err != nil {
// 		t.Fatalf("warmup AXFR: %v", err)
// 	}
// 	rrs, _ := drainTransfer(t, ch)
// 	currentSerial := rrs[0].(*dns.SOA).Serial
//
// 	// Now ask for IXFR with that exact serial.
// 	ch, err = plugin.Transfer("cat.example.com.", currentSerial)
// 	if err != nil {
// 		t.Fatalf("Transfer (IXFR no-op): %v", err)
// 	}
// 	got, _ := drainTransfer(t, ch)
// 	if len(got) != 1 {
// 		t.Errorf(
// 			"IXFR no-op should return 1 RR (single SOA), got %d: %v",
// 			len(got),
// 			got,
// 		)
// 	}
// 	if _, ok := got[0].(*dns.SOA); !ok {
// 		t.Errorf("IXFR no-op RR should be SOA, got %v", got[0])
// 	}
// }

// pollCatalogs end-to-end: poller fetches active zones and parked catalog
// zones in the same cycle and writes a snapshot for the catalog.
// func TestPollOnce_PopulatesCatalogCache(t *testing.T) {
// 	mux, plugin := newMockPlugin(t)
// 	plugin.cache = zonecache.New(8)
// 	mux.HandleFunc(
// 		"/api/plugins/netbox-dns/zones/", catalogZonesHandler(
// 			fixtureActiveZonesForCatalog, fixtureCatalogZone,
// 		),
// 	)
// 	// Records endpoint exists for the active zones path; return empty.
// 	mux.HandleFunc(
// 		"/api/plugins/netbox-dns/records/",
// 		func(w http.ResponseWriter, _ *http.Request) {
// 			writeJSON(w, `{"count":0,"next":null,"previous":null,"results":[]}`)
// 		},
// 	)
//
// 	plugin.pollOnce()
//
// 	snap, ok := plugin.cache.Latest("cat.example.com")
// 	if !ok {
// 		t.Fatal("catalog cache miss after pollOnce")
// 	}
// 	if len(snap.RRs) != 5 {
// 		t.Errorf("snapshot len = %d, want 5", len(snap.RRs))
// 	}
// }
