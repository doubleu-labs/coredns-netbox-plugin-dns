package netboxdns

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/miekg/dns"
)

// recordsHandler returns an http.HandlerFunc that serves the records endpoint
// based on the requested fqdn / name / type / zone_id query parameters. The
// fixtures map's key is "<fqdn|@name>|<type1,type2,...>"; values are full
// JSON envelopes (count + results).
func recordsHandler(t *testing.T, fixtures map[string]string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		key := q.Get("fqdn")
		if key == "" {
			key = "@" + q.Get("name")
		}
		typesKey := ""
		for i, tp := range q["type"] {
			if i > 0 {
				typesKey += ","
			}
			typesKey += tp
		}
		full := key + "|" + typesKey
		if body, ok := fixtures[full]; ok {
			writeJSON(w, body)
			return
		}
		// Default: empty results.
		writeJSON(w, `{"count": 0, "next": null, "previous": null, "results": []}`)
	}
}

// zoneList serves a fixed zone list at /zones/.
func zonesHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, body)
	}
}

const allTestZones = `{
    "count": 2,
    "next": null,
    "previous": null,
    "results": [
        {"id": 1, "name": "example.com", "default_ttl": 3600, "nameservers": [{"name": "ns1.example.com"}, {"name": "ns2.example.com"}]},
        {"id": 2, "name": "sub.example.com", "default_ttl": 3600, "nameservers": [{"name": "ns1.example.com"}]}
    ]
}`

func TestLookup_SOAOrigin(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(allTestZones))
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, map[string]string{
		// processOrigin asks for name=@ type=SOA,NS in zone 1
		"@@|SOA,NS": `{"count": 2, "next": null, "previous": null, "results": [
            {"type": "SOA", "fqdn": "example.com.", "absolute_value": "ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600", "ttl": 86400, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}},
            {"type": "NS", "fqdn": "example.com.", "absolute_value": "ns1.example.com.", "ttl": null, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}}
        ]}`,
		// processExtra(NS) asks for fqdn=ns1.example.com. type=A in zone 1
		"ns1.example.com.|A": `{"count": 1, "next": null, "previous": null, "results": [
            {"type": "A", "fqdn": "ns1.example.com.", "absolute_value": "10.0.0.10", "ttl": null, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}}
        ]}`,
	}))

	resp, err := plugin.lookup("example.com.", dns.TypeSOA, 1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if resp.LookupResult != lookupSuccess {
		t.Fatalf("LookupResult = %v, want lookupSuccess", resp.LookupResult)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("answer len = %d, want 1", len(resp.Answer))
	}
	if _, ok := resp.Answer[0].(*dns.SOA); !ok {
		t.Errorf("answer[0] is not SOA: %#v", resp.Answer[0])
	}
	if len(resp.Ns) != 1 {
		t.Errorf("ns len = %d, want 1", len(resp.Ns))
	}
	if len(resp.Extra) != 1 {
		t.Errorf("extra len = %d, want 1", len(resp.Extra))
	}
}

func TestLookup_DirectA(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(allTestZones))
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, map[string]string{
		// lookupDirect for A,CNAME on web.example.com
		"web.example.com.|A,CNAME": `{"count": 1, "next": null, "previous": null, "results": [
            {"type": "A", "fqdn": "web.example.com.", "absolute_value": "10.0.0.20", "ttl": null, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}}
        ]}`,
	}))

	resp, err := plugin.lookup("web.example.com.", dns.TypeA, 1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if resp.LookupResult != lookupSuccess {
		t.Fatalf("LookupResult = %v, want lookupSuccess", resp.LookupResult)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("answer len = %d, want 1", len(resp.Answer))
	}
	a, ok := resp.Answer[0].(*dns.A)
	if !ok || a.A.String() != "10.0.0.20" {
		t.Errorf("unexpected answer: %#v", resp.Answer[0])
	}
}

func TestLookup_Delegation(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	// Only one zone here so delegation is matched against example.com.
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(`{
        "count": 1, "next": null, "previous": null,
        "results": [
            {"id": 1, "name": "example.com", "default_ttl": 3600, "nameservers": []}
        ]
    }`))
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, map[string]string{
		// lookupDirect (A,CNAME) returns nothing
		"deleg.example.com.|A,CNAME": `{"count": 0, "next": null, "previous": null, "results": []}`,
		// lookupDelegate asks for NS on deleg.example.com
		"deleg.example.com.|NS": `{"count": 1, "next": null, "previous": null, "results": [
            {"type": "NS", "fqdn": "deleg.example.com.", "absolute_value": "ns1.delegated.example.net.", "ttl": 3600, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}}
        ]}`,
	}))

	resp, err := plugin.lookup("deleg.example.com.", dns.TypeA, 1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if resp.LookupResult != lookupDelegation {
		t.Fatalf("LookupResult = %v, want lookupDelegation", resp.LookupResult)
	}
	if len(resp.Ns) != 1 {
		t.Errorf("ns len = %d, want 1", len(resp.Ns))
	}
}

func TestLookup_NXDomain(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(allTestZones))
	// records endpoint returns empty for everything
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, nil))

	resp, err := plugin.lookup("missing.example.com.", dns.TypeA, 1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if resp.LookupResult != lookupNameError {
		t.Errorf("LookupResult = %v, want lookupNameError", resp.LookupResult)
	}
}

func TestLookup_NoMatchingZone(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(allTestZones))
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, nil))

	resp, err := plugin.lookup("example.net.", dns.TypeA, 1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if resp.LookupResult != lookupNameError {
		t.Errorf("LookupResult = %v, want lookupNameError", resp.LookupResult)
	}
}

// TestLookup_PropagatesViewName verifies that NetboxDNS.viewName flows through
// matchZone → GetZones and ends up as ?view=<name> on the /zones/ request.
func TestLookup_PropagatesViewName(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.viewName = "internal"

	var gotView string
	mux.HandleFunc("/api/plugins/netbox-dns/zones/", func(w http.ResponseWriter, r *http.Request) {
		gotView = r.URL.Query().Get("view")
		writeJSON(w, allTestZones)
	})
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, nil))

	if _, err := plugin.lookup("missing.example.com.", dns.TypeA, 1); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if gotView != "internal" {
		t.Errorf("?view query = %q, want %q", gotView, "internal")
	}
}

// Sanity-check that the path the mock plugin uses really looks like the
// production one (catch a future Parse() change quickly).
func TestNewMockPluginURL(t *testing.T) {
	_, plugin := newMockPlugin(t)
	got := plugin.requestClient.NetboxURL.String()
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Path != "/api/plugins/netbox-dns/" {
		t.Errorf("path = %q, want /api/plugins/netbox-dns/", u.Path)
	}
}
