package netbox

import (
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRecordQuery_Encode_FQDNAndType(t *testing.T) {
	q := &RecordQuery{
		FQDN: "web.example.com.",
		Type: []string{"A", "CNAME"},
	}
	got, err := url.ParseQuery(q.Encode())
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if got.Get("fqdn") != "web.example.com." {
		t.Errorf("fqdn = %q", got.Get("fqdn"))
	}
	types := got["type"]
	if len(types) != 2 || types[0] != "A" || types[1] != "CNAME" {
		t.Errorf("type = %v, want [A CNAME]", types)
	}
	if got.Get("zone_id") != "" {
		t.Errorf("zone_id should be empty, got %q", got.Get("zone_id"))
	}
}

func TestRecordQuery_Encode_NameAndZone(t *testing.T) {
	q := &RecordQuery{
		Name: "@",
		Type: []string{"SOA", "NS"},
		Zone: &Zone{ID: 7},
	}
	got, err := url.ParseQuery(q.Encode())
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if got.Get("name") != "@" {
		t.Errorf("name = %q", got.Get("name"))
	}
	if got.Get("zone_id") != "7" {
		t.Errorf("zone_id = %q", got.Get("zone_id"))
	}
	if len(got["type"]) != 2 {
		t.Errorf("expected 2 types, got %v", got["type"])
	}
}

func TestRecordQuery_Encode_Empty(t *testing.T) {
	q := &RecordQuery{}
	if q.Encode() != "" {
		t.Errorf("empty query should encode to empty string, got %q", q.Encode())
	}
}

// fixture: 2 records, one with explicit TTL and one with ttl=null
const fixtureRecordsForZone1 = `{
    "count": 2,
    "next": null,
    "previous": null,
    "results": [
        {
            "type": "A",
            "absolute_value": "10.0.0.10",
            "ttl": 600,
            "zone": {"id": 1, "name": "example.com", "default_ttl": 3600},
            "fqdn": "ns1.example.com."
        },
        {
            "type": "A",
            "absolute_value": "10.0.0.11",
            "ttl": null,
            "zone": {"id": 1, "name": "example.com", "default_ttl": 3600},
            "fqdn": "ns2.example.com."
        }
    ]
}`

func TestGetRecordsQuery_WithZone_FillsTTLFromZone(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("zone_id"); got != "1" {
			t.Errorf("zone_id query = %q, want 1", got)
		}
		writeJSON(w, fixtureRecordsForZone1)
	})

	zone := &Zone{ID: 1, Name: "example.com", DefaultTTL: 3600}
	records, err := GetRecordsQuery(client, &RecordQuery{Zone: zone})
	if err != nil {
		t.Fatalf("GetRecordsQuery: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want 2", len(records))
	}
	if records[0].TTL == nil || *records[0].TTL != 600 {
		t.Errorf("first TTL = %v, want 600", records[0].TTL)
	}
	if records[1].TTL == nil || *records[1].TTL != 3600 {
		t.Errorf("second TTL should fall back to zone default 3600, got %v", records[1].TTL)
	}
}

// When the caller does not pass a Zone, GetRecordsQuery must call
// resolveRecordTTLs which fetches /zones/<id>/ once per unique zone.
func TestGetRecordsQuery_WithoutZone_ResolvesAndCachesTTL(t *testing.T) {
	_, mux, client := mockNetbox(t)
	var zoneFetches int32

	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureRecordsForZone1)
	})
	mux.HandleFunc("/api/plugins/netbox-dns/zones/1/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&zoneFetches, 1)
		writeJSON(w, fixtureZoneByID1)
	})

	records, err := GetRecordsQuery(client, &RecordQuery{FQDN: "ns2.example.com."})
	if err != nil {
		t.Fatalf("GetRecordsQuery: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want 2", len(records))
	}
	// First record had explicit ttl=600, must remain unchanged.
	if records[0].TTL == nil || *records[0].TTL != 600 {
		t.Errorf("first TTL = %v, want 600", records[0].TTL)
	}
	if records[1].TTL == nil || *records[1].TTL != 3600 {
		t.Errorf("second TTL = %v, want 3600", records[1].TTL)
	}
	if got := atomic.LoadInt32(&zoneFetches); got != 1 {
		t.Errorf("zone fetched %d times, want exactly 1 (cache hit on second null TTL)", got)
	}
}

func TestResolveRecordTTLs_MultipleZones(t *testing.T) {
	_, mux, client := mockNetbox(t)

	const fixtureZone2 = `{"id": 2, "name": "example.org", "default_ttl": 7200, "nameservers": []}`

	var fetches int32
	mux.HandleFunc("/api/plugins/netbox-dns/zones/1/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&fetches, 1)
		writeJSON(w, fixtureZoneByID1)
	})
	mux.HandleFunc("/api/plugins/netbox-dns/zones/2/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&fetches, 1)
		writeJSON(w, fixtureZone2)
	})

	records := []Record{
		{Type: "A", FQDN: "a.example.com.", Zone: Zone{ID: 1}},
		{Type: "A", FQDN: "b.example.com.", Zone: Zone{ID: 1}}, // cache hit
		{Type: "A", FQDN: "x.example.org.", Zone: Zone{ID: 2}}, // new fetch
	}
	out, err := resolveRecordTTLs(client, records)
	if err != nil {
		t.Fatalf("resolveRecordTTLs: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	if *out[0].TTL != 3600 || *out[1].TTL != 3600 || *out[2].TTL != 7200 {
		t.Errorf("ttls = %v %v %v, want 3600 3600 7200",
			*out[0].TTL, *out[1].TTL, *out[2].TTL)
	}
	if got := atomic.LoadInt32(&fetches); got != 2 {
		t.Errorf("zone fetches = %d, want 2 (one per unique zone id)", got)
	}
}

func TestResolveRecordTTLs_SkipsRecordsWithExplicitTTL(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc("/api/plugins/netbox-dns/zones/", func(w http.ResponseWriter, r *http.Request) {
		// Should never be called.
		t.Errorf("unexpected zone fetch: %s", r.URL.Path)
		http.NotFound(w, r)
	})
	records := []Record{{Type: "A", FQDN: "x.example.com.", Zone: Zone{ID: 1}, TTL: new(uint32(99))}}
	out, err := resolveRecordTTLs(client, records)
	if err != nil {
		t.Fatalf("resolveRecordTTLs: %v", err)
	}
	if *out[0].TTL != 99 {
		t.Errorf("ttl mutated, got %d", *out[0].TTL)
	}
}

func TestGetRecordsQuery_APIError(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	_, err := GetRecordsQuery(client, &RecordQuery{Zone: &Zone{ID: 1}})
	if err == nil || !strings.Contains(err.Error(), "request error") {
		t.Fatalf("want request error, got %v", err)
	}
}
