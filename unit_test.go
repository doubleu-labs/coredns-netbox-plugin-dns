package netboxdns

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// newMockPlugin returns a NetboxDNS wired against an httptest.Server. Tests
// register their handlers on the returned mux. The plugin's NetboxURL points
// at <server>/api/plugins/netbox-dns/ to mirror what parse.go produces in
// production.
func newMockPlugin(t *testing.T) (*http.ServeMux, *NetboxDNS) {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL + "/api/plugins/netbox-dns/")
	if err != nil {
		t.Fatalf("parse mock url: %v", err)
	}
	plugin := &NetboxDNS{
		zones: []string{"."},
		requestClient: &netbox.Client{
			Client:    srv.Client(),
			NetboxURL: base,
			UserAgent: "netboxdns-unit-tests",
		},
		catalogTracker: newCatalogTracker(),
	}
	plugin.requestClient.SetToken("test-token")
	return mux, plugin
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// ----- fixQType ---------------------------------------------------------

func TestFixQType(t *testing.T) {
	cases := []struct {
		name   string
		qtype  uint16
		family int
		want   uint16
	}{
		{"A over v4", dns.TypeA, 1, dns.TypeA},
		{"A over v6", dns.TypeA, 2, dns.TypeAAAA},
		{"AAAA over v4", dns.TypeAAAA, 1, dns.TypeA},
		{"AAAA over v6", dns.TypeAAAA, 2, dns.TypeAAAA},
		{"MX unchanged v4", dns.TypeMX, 1, dns.TypeMX},
		{"TXT unchanged v6", dns.TypeTXT, 2, dns.TypeTXT},
		{"NS unchanged", dns.TypeNS, 1, dns.TypeNS},
	}
	for _, tc := range cases {
		t.Run(
			tc.name, func(t *testing.T) {
				got := fixQType(tc.qtype, tc.family)
				if got != tc.want {
					t.Errorf(
						"fixQType(%d, %d) = %d, want %d",
						tc.qtype, tc.family, got, tc.want,
					)
				}
			},
		)
	}
}

// ----- recordsToRR / recordToTXT / filterRRByType -----------------------

func ttlPtr(v uint32) *uint32 { return new(v) }

func TestRecordsToRR_BasicTypes(t *testing.T) {
	records := []netbox.Record{
		{
			Type:          "A",
			FQDN:          "host.example.com.",
			AbsoluteValue: "10.0.0.1",
			TTL:           ttlPtr(60),
		},
		{
			Type:          "AAAA",
			FQDN:          "host.example.com.",
			AbsoluteValue: "2001:db8::1",
			TTL:           ttlPtr(60),
		},
		{
			Type:          "CNAME",
			FQDN:          "www.example.com.",
			AbsoluteValue: "host.example.com.",
			TTL:           ttlPtr(300),
		},
		{
			Type:          "MX",
			FQDN:          "example.com.",
			AbsoluteValue: "10 mail.example.com.",
			TTL:           ttlPtr(300),
		},
		{
			Type:          "NS",
			FQDN:          "example.com.",
			AbsoluteValue: "ns1.example.com.",
			TTL:           ttlPtr(3600),
		},
		{
			Type:          "PTR",
			FQDN:          "1.0.0.10.in-addr.arpa.",
			AbsoluteValue: "host.example.com.",
			TTL:           ttlPtr(3600),
		},
		{
			Type:          "SRV",
			FQDN:          "_sip._tcp.example.com.",
			AbsoluteValue: "10 20 5060 sip.example.com.",
			TTL:           ttlPtr(300),
		},
	}
	rrs, err := recordsToRR(records)
	if err != nil {
		t.Fatalf("recordsToRR: %v", err)
	}
	if len(rrs) != len(records) {
		t.Fatalf("len = %d, want %d", len(rrs), len(records))
	}

	if a, ok := rrs[0].(*dns.A); !ok || a.A.String() != "10.0.0.1" {
		t.Errorf("A record wrong: %#v", rrs[0])
	}
	if aaaa, ok := rrs[1].(*dns.AAAA); !ok || aaaa.AAAA.String() != "2001:db8::1" {
		t.Errorf("AAAA record wrong: %#v", rrs[1])
	}
	if c, ok := rrs[2].(*dns.CNAME); !ok || c.Target != "host.example.com." {
		t.Errorf("CNAME wrong: %#v", rrs[2])
	}
	if mx, ok := rrs[3].(*dns.MX); !ok || mx.Preference != 10 || mx.Mx != "mail.example.com." {
		t.Errorf("MX wrong: %#v", rrs[3])
	}
	if ns, ok := rrs[4].(*dns.NS); !ok || ns.Ns != "ns1.example.com." {
		t.Errorf("NS wrong: %#v", rrs[4])
	}
	if ptr, ok := rrs[5].(*dns.PTR); !ok || ptr.Ptr != "host.example.com." {
		t.Errorf("PTR wrong: %#v", rrs[5])
	}
	if srv, ok := rrs[6].(*dns.SRV); !ok || srv.Port != 5060 || srv.Target != "sip.example.com." {
		t.Errorf("SRV wrong: %#v", rrs[6])
	}
}

func TestRecordsToRR_SOA(t *testing.T) {
	records := []netbox.Record{
		{
			Type:          "SOA",
			FQDN:          "example.com.",
			AbsoluteValue: "ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600",
			TTL:           ttlPtr(86400),
		},
	}
	rrs, err := recordsToRR(records)
	if err != nil {
		t.Fatalf("recordsToRR: %v", err)
	}
	soa, ok := rrs[0].(*dns.SOA)
	if !ok {
		t.Fatalf("not a SOA: %#v", rrs[0])
	}
	if soa.Ns != "ns1.example.com." || soa.Mbox != "admin.example.com." || soa.Serial != 1 {
		t.Errorf("SOA fields wrong: %+v", soa)
	}
}

func TestRecordsToRR_InvalidValue(t *testing.T) {
	records := []netbox.Record{
		{
			Type:          "A",
			FQDN:          "bad.example.com.",
			AbsoluteValue: "not-an-ip",
			TTL:           ttlPtr(60),
		},
	}
	_, err := recordsToRR(records)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestRecordsToRR_EmptyInput(t *testing.T) {
	rrs, err := recordsToRR(nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(rrs) != 0 {
		t.Errorf("expected empty, got %d", len(rrs))
	}
}

func TestRecordToTXT_SimpleAndMultiValue(t *testing.T) {
	cases := []struct {
		name string
		val  string
		want []string
	}{
		{"plain", "hello world", []string{"hello world"}},
		{"single quoted", `"hello"`, []string{"hello"}},
		{
			"multi value", `"v=spf1" "include:_spf.example.com" "-all"`,
			[]string{"v=spf1", "include:_spf.example.com", "-all"},
		},
		{
			"with literal newlines stripped",
			`"line1\r\nline2"`,
			[]string{"line1line2"},
		},
	}
	for _, tc := range cases {
		t.Run(
			tc.name, func(t *testing.T) {
				rec := netbox.Record{
					Type:          "TXT",
					FQDN:          "txt.example.com.",
					AbsoluteValue: tc.val,
					TTL:           ttlPtr(60),
				}
				rr := recordToTXT(rec)
				if len(rr.Txt) != len(tc.want) {
					t.Fatalf(
						"len = %d, want %d (%v)",
						len(rr.Txt),
						len(tc.want),
						rr.Txt,
					)
				}
				for i := range tc.want {
					if rr.Txt[i] != tc.want[i] {
						t.Errorf(
							"Txt[%d] = %q, want %q",
							i,
							rr.Txt[i],
							tc.want[i],
						)
					}
				}
			},
		)
	}
}

func TestFilterRRByType(t *testing.T) {
	rrs := []dns.RR{
		&dns.A{Hdr: dns.RR_Header{Rrtype: dns.TypeA}},
		&dns.NS{Hdr: dns.RR_Header{Rrtype: dns.TypeNS}},
		&dns.A{Hdr: dns.RR_Header{Rrtype: dns.TypeA}},
		&dns.AAAA{Hdr: dns.RR_Header{Rrtype: dns.TypeAAAA}},
	}
	a := filterRRByType(rrs, dns.TypeA)
	if len(a) != 2 {
		t.Errorf("filtered A count = %d, want 2", len(a))
	}
	none := filterRRByType(nil, dns.TypeA)
	if len(none) != 0 {
		t.Errorf("nil input -> non-empty: %v", none)
	}
}

// ----- matchZone via mock -----------------------------------------------

func TestMatchZone(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(
				w, `{
            "count": 3,
            "next": null,
            "previous": null,
            "results": [
                {"id": 1, "name": "example.com", "default_ttl": 3600, "nameservers": []},
                {"id": 2, "name": "sub.example.com", "default_ttl": 3600, "nameservers": []},
                {"id": 3, "name": "example.org", "default_ttl": 3600, "nameservers": []}
            ]
        }`,
			)
		},
	)

	cases := []struct {
		name    string
		qname   string
		wantNil bool
		want    string
	}{
		{"exact match", "example.com", false, "example.com"},
		{"subdomain to parent", "host.example.com", false, "example.com"},
		{"longest match wins", "a.sub.example.com", false, "sub.example.com"},
		{"other tld", "example.net", true, ""},
	}
	for _, tc := range cases {
		t.Run(
			tc.name, func(t *testing.T) {
				z, err := plugin.matchZone(tc.qname)
				if err != nil {
					t.Fatalf("matchZone: %v", err)
				}
				if tc.wantNil {
					if z != nil {
						t.Errorf("want nil, got %+v", z)
					}
					return
				}
				if z == nil || z.Name != tc.want {
					t.Errorf("got %+v, want %q", z, tc.want)
				}
			},
		)
	}
}

func TestMatchZone_APIError(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		},
	)
	if _, err := plugin.matchZone("example.com"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
