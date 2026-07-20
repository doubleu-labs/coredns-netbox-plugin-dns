package netboxdns

import (
	"bytes"
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

// TestTransfer_AXFR performs a live AXFR request against a fully functional
// CoreDNS server with NetboxDNS and transfer plugins. This test confirms that
// the AXFR response is the correct format and the expected and received records
// match byte-for-byte.
func TestTransfer_AXFR(t *testing.T) {
	//noinspection LongLine
	testCase := test.Case{
		Qname: "example.com.",
		Qtype: dns.TypeAXFR,
		Answer: []dns.RR{
			test.SOA("example.com. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			test.MX("example.com. 3600 IN MX 10 mail.example.com."),
			test.NS("example.com. 3600 IN NS ns1.example.com."),
			test.NS("example.com. 3600 IN NS ns2.example.com."),
			test.TXT(`example.com. 3600 IN TXT "my value" "second my value" "third my value"`),
			test.TXT(`example.com. 3600 IN TXT "newline record" "second value"`),
			test.TXT(`example.com. 3600 IN TXT "some value" "another value"`),
			test.TXT(`example.com. 3600 IN TXT "v=DMARC1;p=none;sp=quarantine;pct=100;rua=admin@example.com;"`),
			test.TXT(`example.com. 3600 IN TXT "v=spf1 ip4:10.0.0.13 ip6:2001:db8:dead:beef::1:13 a -all"`),
			test.A("mail.example.com. 3600 IN A 10.0.1.13"),
			test.AAAA("mail.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:13"),
			test.A("ns1.example.com. 3600 IN A 10.0.1.10"),
			test.AAAA("ns1.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:10"),
			test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
			test.AAAA("ns2.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:11"),
			test.NS("sub.example.com. 3600 IN NS ns1.sub.example.com."),
			test.NS("sub.example.com. 3600 IN NS ns2.sub.example.com."),
			test.A("web.example.com. 3600 IN A 10.0.1.17"),
			test.AAAA("web.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:17"),
			test.CNAME("www.example.com. 3600 IN CNAME web.example.com."),
			test.SOA("example.com. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
		},
	}

	corefile := `
	transfer . {
		to *
	}
	netboxdns {
		token <!TOKEN>
		url <!URL>
		zone_poller
	}`
	isTransferTest = true
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	msg := new(dns.Msg)
	msg.SetAxfr(testCase.Qname)

	envelopeChan, err := server.Transfer(msg)
	if err != nil {
		t.Fatalf("failed to initiate live transfer: %v", err)
	}

	combinedResponse := new(dns.Msg)
	combinedResponse.SetReply(msg)

	for envelope := range envelopeChan {
		if envelope.Error != nil {
			t.Fatalf("error in axfr envelope stream: %v", envelope.Error)
		}
		combinedResponse.Answer = append(
			combinedResponse.Answer,
			envelope.RR...,
		)
	}

	if len(combinedResponse.Answer) < 2 {
		t.Fatalf(
			"axfr response contains too few records: got %d",
			len(combinedResponse.Answer),
		)
	}

	firstRR := combinedResponse.Answer[0]
	lastRR := combinedResponse.Answer[len(combinedResponse.Answer)-1]

	if firstRR.Header().Rrtype != dns.TypeSOA {
		t.Errorf(
			"protocol violation: first record is not SOA; got %s",
			dns.TypeToString[firstRR.Header().Rrtype],
		)
	}
	if lastRR.Header().Rrtype != dns.TypeSOA {
		t.Errorf(
			"protocol violation: last record is not SOA; got %s",
			dns.TypeToString[lastRR.Header().Rrtype],
		)
	}
	if firstRR.String() != lastRR.String() {
		t.Errorf(
			"opening and closing soa records do not match: "+
				"opening: %s; closing: %s",
			firstRR.String(),
			lastRR.String(),
		)
	}

	// strip the opening and closing soa records since they were already checked
	wantInner := testCase.Answer[1 : len(testCase.Answer)-1]
	gotInner := combinedResponse.Answer[1 : len(combinedResponse.Answer)-1]

	if len(gotInner) != len(wantInner) {
		t.Errorf(
			"axfr response contains wrong number of records: "+
				"got %d; want %d",
			len(gotInner),
			len(wantInner),
		)
	}

	// do a byte-by-byte comparison of the inner zone records
	for i := range gotInner {
		gotBytes := make([]byte, dns.MaxMsgSize)
		wantBytes := make([]byte, dns.MaxMsgSize)
		_, gotErr := dns.PackRR(gotInner[i], gotBytes, 0, nil, false)
		_, wantErr := dns.PackRR(wantInner[i], wantBytes, 0, nil, false)
		if gotErr != nil || wantErr != nil {
			t.Errorf(
				"failed to pack record at index %d: got %s; want %s",
				i,
				gotErr,
				wantErr,
			)
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf(
				"record mismatch at index %d: got %s; want %s",
				i,
				gotInner[i].String(),
				wantInner[i].String(),
			)
		}
	}
}
