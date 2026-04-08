package netboxdns

import (
	"context"
	"net/http"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

// TestServeDNS_SuccessSOA exercises the full ServeDNS handler against a mock
// NetBox: a SOA query for the zone apex must produce an authoritative answer
// with rcode NoError.
func TestServeDNS_SuccessSOA(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.Next = test.ErrorHandler()

	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(`{
        "count": 1, "next": null, "previous": null,
        "results": [
            {"id": 1, "name": "example.com", "default_ttl": 3600, "nameservers": [{"name": "ns1.example.com"}]}
        ]
    }`))
	mux.Handle("/api/plugins/netbox-dns/records/", recordsHandler(t, map[string]string{
		"@@|SOA,NS": `{"count": 2, "next": null, "previous": null, "results": [
            {"type": "SOA", "fqdn": "example.com.", "absolute_value": "ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600", "ttl": 86400, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}},
            {"type": "NS", "fqdn": "example.com.", "absolute_value": "ns1.example.com.", "ttl": null, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}}
        ]}`,
		"ns1.example.com.|A": `{"count": 1, "next": null, "previous": null, "results": [
            {"type": "A", "fqdn": "ns1.example.com.", "absolute_value": "10.0.0.10", "ttl": null, "zone": {"id": 1, "name": "example.com", "default_ttl": 3600}}
        ]}`,
	}))

	tc := test.Case{Qname: "example.com.", Qtype: dns.TypeSOA}
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	rcode, err := plugin.ServeDNS(context.Background(), rec, tc.Msg())
	if err != nil {
		t.Fatalf("ServeDNS error: %v", err)
	}
	if rcode != dns.RcodeSuccess {
		t.Errorf("rcode = %d, want %d", rcode, dns.RcodeSuccess)
	}
	if rec.Msg == nil {
		t.Fatal("nil response message")
	}
	if !rec.Msg.Authoritative {
		t.Errorf("expected authoritative reply")
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("answer len = %d, want 1", len(rec.Msg.Answer))
	}
	if _, ok := rec.Msg.Answer[0].(*dns.SOA); !ok {
		t.Errorf("answer[0] not SOA: %#v", rec.Msg.Answer[0])
	}
}

// TestServeDNS_NXDomain exercises the NXDOMAIN branch of ServeDNS: the zone
// matches but no records exist for the queried name.
func TestServeDNS_NXDomain(t *testing.T) {
	mux, plugin := newMockPlugin(t)
	plugin.Next = test.ErrorHandler()

	mux.Handle("/api/plugins/netbox-dns/zones/", zonesHandler(`{
        "count": 1, "next": null, "previous": null,
        "results": [
            {"id": 1, "name": "example.com", "default_ttl": 3600, "nameservers": []}
        ]
    }`))
	mux.HandleFunc("/api/plugins/netbox-dns/records/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"count": 0, "next": null, "previous": null, "results": []}`)
	})

	tc := test.Case{Qname: "missing.example.com.", Qtype: dns.TypeA}
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	rcode, err := plugin.ServeDNS(context.Background(), rec, tc.Msg())
	if err != nil {
		t.Fatalf("ServeDNS error: %v", err)
	}
	if rcode != dns.RcodeSuccess {
		t.Errorf("rcode = %d, want %d (rcode is in the message)", rcode, dns.RcodeSuccess)
	}
	if rec.Msg == nil {
		t.Fatal("nil response message")
	}
	if rec.Msg.Rcode != dns.RcodeNameError {
		t.Errorf("msg.Rcode = %d, want NXDOMAIN (%d)", rec.Msg.Rcode, dns.RcodeNameError)
	}
	if !rec.Msg.Authoritative {
		t.Errorf("NXDOMAIN should still be authoritative")
	}
}
