package netboxdns

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

func TestPluginName(t *testing.T) {
	netboxdns := &NetboxDNS{}
	name := netboxdns.Name()
	if name != pluginName {
		t.Errorf("plugin name is wrong. how did this happen...")
	}
}

type testFamily int

const (
	testFamilyV4 testFamily = iota
	testFamilyV6
)

var testFamilyToString map[testFamily]string = map[testFamily]string{
	testFamilyV4: "V4",
	testFamilyV6: "V6",
}

const (
	testInstanceToken   string = "w5pgWXPqZVmngLN4w4XwuPvZfUC72ytDxnnHgEmI"
	testInstanceUrlHost string = "localhost:9999"
	testInstanceUrlPath string = "/api/plugins/netbox-dns/"
)

var netboxdnsPlugin NetboxDNS = NetboxDNS{
	Next:  test.ErrorHandler(),
	zones: []string{"."},
	requestClient: &netbox.APIRequestClient{
		Client: &http.Client{
			Timeout: time.Second * 30,
		},
		NetboxURL: &url.URL{
			Scheme: "http",
			Host:   testInstanceUrlHost,
			Path:   testInstanceUrlPath,
		},
		Token: testInstanceToken,
	},
}

func RunTestLookup(t *testing.T, tcs []test.Case, family testFamily) {
	for _, tc := range tcs {
		tcName := fmt.Sprintf(
			"%s %s %s",
			tc.Qname,
			dns.TypeToString[tc.Qtype],
			testFamilyToString[family],
		)
		t.Run(tcName, func(t *testing.T) {
			msg := tc.Msg()
			respWriter := GetTestResponseWriter(family)
			rec := dnstest.NewRecorder(respWriter)
			_, err := netboxdnsPlugin.ServeDNS(context.Background(), rec, msg)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}
			resp := rec.Msg
			if resp == nil {
				t.Fatal("got nil response message")
			}
			if ok := RunTestLookupCheckCNAME(t, tc, resp); !ok {
				return
			}
			if err := test.SortAndCheck(resp, tc); err != nil {
				t.Logf("%s\n", rec.Msg)
				t.Error(err)
			}
		})
	}
}

func GetTestResponseWriter(family testFamily) dns.ResponseWriter {
	var respWriter dns.ResponseWriter
	switch family {
	case testFamilyV4:
		respWriter = &test.ResponseWriter{}
	case testFamilyV6:
		respWriter = &test.ResponseWriter6{}
	}
	return respWriter
}

func RunTestLookupCheckCNAME(t *testing.T, tc test.Case, resp *dns.Msg) bool {
	if err := test.CNAMEOrder(resp); err != nil {
		t.Errorf("cname response out of order")
	}
	if tc.Qtype == dns.TypeCNAME || RunTestLookupContainsCNAME(resp) {
		if err := test.Header(tc, resp); err != nil {
			t.Error(err)
		}
		if err := test.Section(tc, test.Answer, resp.Answer); err != nil {
			t.Error(err)
		}
		if err := test.Section(tc, test.Ns, resp.Ns); err != nil {
			t.Error(err)
		}
		if err := test.Section(tc, test.Extra, resp.Extra); err != nil {
			t.Error(err)
		}
		return false
	}
	return true
}

func RunTestLookupContainsCNAME(resp *dns.Msg) bool {
	out := false
	for _, rr := range resp.Answer {
		if rr.Header().Rrtype == dns.TypeCNAME {
			out = true
		}
	}
	return out
}

var (
	exampledotcomName          string = "example.com."
	subdotexampledotcomName    string = "sub.example.com."
	subtwodotexampledotcomName string = "subtwo.example.com."

	exampledotcomNS []dns.RR = []dns.RR{
		test.NS("example.com. 3600 IN NS dns01.example.com."),
		test.NS("example.com. 3600 IN NS dns02.example.com."),
	}
	subdotexampledotcomNS []dns.RR = []dns.RR{
		test.NS("sub.example.com. 3600 IN NS dns01.example.com."),
		test.NS("sub.example.com. 3600 IN NS dns02.example.com."),
	}
	subtwodotexampledotcomNS []dns.RR = []dns.RR{
		test.NS("subtwo.example.com. 3600 IN NS dns01.example.com."),
		test.NS("subtwo.example.com. 3600 IN NS dns02.example.com."),
	}
	exampledotcomNS1Record4 dns.RR   = test.A("dns01.example.com. 3600 IN A 10.0.0.10")
	exampledotcomNS2Record4 dns.RR   = test.A("dns02.example.com. 3600 IN A 10.0.0.11")
	exampledotcomNSAddr4    []dns.RR = []dns.RR{
		exampledotcomNS1Record4,
		exampledotcomNS2Record4,
	}
	exampledotcomNS1Record6 dns.RR   = test.AAAA("dns01.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:10")
	exampledotcomNS2Record6 dns.RR   = test.AAAA("dns02.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:11")
	exampledotcomNSAddr6    []dns.RR = []dns.RR{
		exampledotcomNS1Record6,
		exampledotcomNS2Record6,
	}

	webdotexampledotcomName        string = "web.example.com."
	webdotexampledotcomRecordA     dns.RR = test.A("web.example.com. 3600 IN A 10.0.0.17")
	webdotexampledotcomRecordAAAA  dns.RR = test.AAAA("web.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:17")
	wwwdotexampledotcomName        string = "www.example.com."
	wwwdotexampledotcomRecordCNAME dns.RR = test.CNAME("www.example.com. 3600 IN CNAME web.example.com.")
)

var (
	testLookupForwardZonesCasesv4 []test.Case = []test.Case{
		{
			Qname: exampledotcomName, Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("example.com. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns:    exampledotcomNS,
			Extra: exampledotcomNSAddr4,
		},
		{
			Qname: subdotexampledotcomName, Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("sub.example.com. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("sub.example.com. 3600 IN NS dns01.example.com"),
				test.NS("sub.example.com. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr4,
		},
		{
			Qname: subtwodotexampledotcomName, Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("subtwo.example.com. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("subtwo.example.com. 3600 IN NS dns01.example.com"),
				test.NS("subtwo.example.com. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr4,
		},
	}

	testLookupReverseZonesCasesv4 []test.Case = []test.Case{
		{
			Qname: "0.0.10.in-addr.arpa.", Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("0.0.10.in-addr.arpa. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("0.0.10.in-addr.arpa. 3600 IN NS dns01.example.com"),
				test.NS("0.0.10.in-addr.arpa. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr4,
		},
		{
			Qname: "1.0.10.in-addr.arpa.", Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("1.0.10.in-addr.arpa. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("1.0.10.in-addr.arpa. 3600 IN NS dns01.example.com"),
				test.NS("1.0.10.in-addr.arpa. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr4,
		},
		{
			Qname: "2.0.10.in-addr.arpa.", Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("2.0.10.in-addr.arpa. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("2.0.10.in-addr.arpa. 3600 IN NS dns01.example.com"),
				test.NS("2.0.10.in-addr.arpa. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr4,
		},
	}

	testLookupForwardZonesCasesv6 []test.Case = []test.Case{
		{
			Qname: exampledotcomName, Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("example.com. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns:    exampledotcomNS,
			Extra: exampledotcomNSAddr6,
		},
		{
			Qname: subdotexampledotcomName, Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("sub.example.com. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("sub.example.com. 3600 IN NS dns01.example.com"),
				test.NS("sub.example.com. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr6,
		},
		{
			Qname: subtwodotexampledotcomName, Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("subtwo.example.com. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("subtwo.example.com. 3600 IN NS dns01.example.com"),
				test.NS("subtwo.example.com. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr6,
		},
	}

	testLookupReverseZonesCasesv6 []test.Case = []test.Case{
		{
			Qname: "1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS dns01.example.com"),
				test.NS("1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr6,
		},
		{
			Qname: "2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS dns01.example.com"),
				test.NS("2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr6,
		},
		{
			Qname: "3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypeSOA,
			Answer: []dns.RR{
				test.SOA("3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA dns01.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
			},
			Ns: []dns.RR{
				test.NS("3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS dns01.example.com"),
				test.NS("3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS dns02.example.com"),
			},
			Extra: exampledotcomNSAddr6,
		},
	}
)

func TestLookupZones(t *testing.T) {
	RunTestLookup(t, testLookupForwardZonesCasesv4, testFamilyV4)
	RunTestLookup(t, testLookupReverseZonesCasesv4, testFamilyV4)
	RunTestLookup(t, testLookupForwardZonesCasesv6, testFamilyV6)
	RunTestLookup(t, testLookupReverseZonesCasesv6, testFamilyV6)
}

var (
	testLookupRecordV4 []test.Case = []test.Case{
		{
			Qname: exampledotcomName, Qtype: dns.TypeNS,
			Answer: exampledotcomNS,
			Extra:  exampledotcomNSAddr4,
		},
		{
			Qname: "dns01.example.com", Qtype: dns.TypeA,
			Answer: []dns.RR{
				exampledotcomNS1Record4,
			},
		},
		{
			Qname: "dns02.example.com", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("dns02.example.com. 3600 IN A 10.0.0.11"),
			},
		},
		{
			Qname: exampledotcomName, Qtype: dns.TypeA,
			Ns:    exampledotcomNS,
			Extra: exampledotcomNSAddr4,
		},
		{
			Qname: "aservice.example.com.", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("aservice.example.com. 3600 IN A 10.0.0.12"),
			},
		},
		{
			Qname: exampledotcomName, Qtype: dns.TypeMX,
			Answer: []dns.RR{
				test.MX("example.com. 3600 IN MX 10 mail.example.com."),
			},
			Extra: []dns.RR{
				test.A("mail.example.com. 3600 IN A 10.0.0.13"),
			},
		},
		{
			Qname: "mail.example.com.", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("mail.example.com. 3600 IN A 10.0.0.13"),
			},
		},
		{
			Qname: exampledotcomName, Qtype: dns.TypeTXT,
			Answer: []dns.RR{
				test.TXT(`example.com. 3600 IN TXT "my value" "second my value" "third my value"`),
				test.TXT(`example.com. 3600 IN TXT "newline record" "second value"`),
				test.TXT(`example.com. 3600 IN TXT "some value" "another value"`),
				test.TXT(`example.com. 3600 IN TXT "v=DMARC1;p=none;sp=quarantine;pct=100;rua=admin@example.com;"`),
				test.TXT(`example.com. 3600 IN TXT "v=spf1 ip4:10.0.0.13 ip6:2001:db8:dead:beef::1:13 a -all"`),
			},
		},
		{
			Qname: "puppet-server-a.example.com.", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("puppet-server-a.example.com. 3600 IN A 10.0.0.15"),
			},
		},
		{
			Qname: "puppet-server-b.example.com.", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("puppet-server-b.example.com. 3600 IN A 10.0.0.16"),
			},
		},
		{
			Qname: "_x-puppet._tcp.example.com.", Qtype: dns.TypeSRV,
			Answer: []dns.RR{
				test.SRV("_x-puppet._tcp.example.com. 3600 IN SRV 0 5 8140 puppet-server-a.example.com."),
				test.SRV("_x-puppet._tcp.example.com. 3600 IN SRV 0 5 8140 puppet-server-b.example.com."),
			},
			Extra: []dns.RR{
				test.A("puppet-server-a.example.com. 3600 IN A 10.0.0.15"),
				test.A("puppet-server-b.example.com. 3600 IN A 10.0.0.16"),
			},
		},
		{
			Qname: webdotexampledotcomName, Qtype: dns.TypeA,
			Answer: []dns.RR{
				webdotexampledotcomRecordA,
			},
		},
		{
			Qname: wwwdotexampledotcomName, Qtype: dns.TypeCNAME,
			Answer: []dns.RR{
				wwwdotexampledotcomRecordCNAME,
				webdotexampledotcomRecordA,
			},
		},
		{
			Qname: wwwdotexampledotcomName, Qtype: dns.TypeA,
			Answer: []dns.RR{
				wwwdotexampledotcomRecordCNAME,
				webdotexampledotcomRecordA,
			},
		},
		{
			Qname: subdotexampledotcomName, Qtype: dns.TypeNS,
			Answer: subdotexampledotcomNS,
			Extra:  exampledotcomNSAddr4,
		},
		{
			Qname: "myservice.sub.example.com.", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("myservice.sub.example.com. 3600 IN A 10.0.1.10"),
			},
		},
		{
			Qname: subtwodotexampledotcomName, Qtype: dns.TypeNS,
			Answer: subtwodotexampledotcomNS,
			Extra:  exampledotcomNSAddr4,
		},
		{
			Qname: "myotherservice.subtwo.example.com.", Qtype: dns.TypeA,
			Answer: []dns.RR{
				test.A("myotherservice.subtwo.example.com. 3600 IN A 10.0.2.10"),
			},
		},
	}

	testLookupPTRV4 []test.Case = []test.Case{
		{
			Qname: "10.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("10.0.0.10.in-addr.arpa. 3600 IN PTR dns01.example.com."),
			},
		},
		{
			Qname: "11.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("11.0.0.10.in-addr.arpa. 3600 IN PTR dns02.example.com."),
			},
		},
		{
			Qname: "12.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("12.0.0.10.in-addr.arpa. 3600 IN PTR aservice.example.com."),
			},
		},
		{
			Qname: "13.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("13.0.0.10.in-addr.arpa. 3600 IN PTR mail.example.com."),
			},
		},
		{
			Qname: "15.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("15.0.0.10.in-addr.arpa. 3600 IN PTR puppet-server-a.example.com."),
			},
		},
		{
			Qname: "16.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("16.0.0.10.in-addr.arpa. 3600 IN PTR puppet-server-b.example.com."),
			},
		},
		{
			Qname: "17.0.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("17.0.0.10.in-addr.arpa. 3600 IN PTR web.example.com."),
			},
		},
		{
			Qname: "10.1.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("10.1.0.10.in-addr.arpa. 3600 IN PTR myservice.sub.example.com."),
			},
		},
		{
			Qname: "10.2.0.10.in-addr.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("10.2.0.10.in-addr.arpa. 3600 IN PTR myotherservice.subtwo.example.com."),
			},
		},
	}

	testLookupRecordV6 []test.Case = []test.Case{
		{
			Qname: exampledotcomName, Qtype: dns.TypeNS,
			Answer: exampledotcomNS,
			Extra:  exampledotcomNSAddr6,
		},
		{
			Qname: "dns01.example.com", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				exampledotcomNS1Record6,
			},
		},
		{
			Qname: "dns02.example.com", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				exampledotcomNS2Record6,
			},
		},
		{
			Qname: exampledotcomName, Qtype: dns.TypeAAAA,
			Ns:    exampledotcomNS,
			Extra: exampledotcomNSAddr6,
		},
		{
			Qname: "aservice.example.com.", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				test.AAAA("aservice.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:12"),
			},
		},
		{
			Qname: exampledotcomName, Qtype: dns.TypeMX,
			Answer: []dns.RR{
				test.MX("example.com. 3600 IN MX 10 mail.example.com."),
			},
			Extra: []dns.RR{
				test.AAAA("mail.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:13"),
			},
		},
		{
			Qname: "mail.example.com.", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				test.AAAA("mail.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:13"),
			},
		},
		{
			Qname: exampledotcomName, Qtype: dns.TypeTXT,
			Answer: []dns.RR{
				test.TXT(`example.com. 3600 IN TXT "my value" "second my value" "third my value"`),
				test.TXT(`example.com. 3600 IN TXT "newline record" "second value"`),
				test.TXT(`example.com. 3600 IN TXT "some value" "another value"`),
				test.TXT(`example.com. 3600 IN TXT "v=DMARC1;p=none;sp=quarantine;pct=100;rua=admin@example.com;"`),
				test.TXT(`example.com. 3600 IN TXT "v=spf1 ip4:10.0.0.13 ip6:2001:db8:dead:beef::1:13 a -all"`),
			},
		},
		{
			Qname: "puppet-server-a.example.com.", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				test.AAAA("puppet-server-a.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:15"),
			},
		},
		{
			Qname: "puppet-server-b.example.com.", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				test.AAAA("puppet-server-b.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:16"),
			},
		},
		{
			Qname: "_x-puppet._tcp.example.com.", Qtype: dns.TypeSRV,
			Answer: []dns.RR{
				test.SRV("_x-puppet._tcp.example.com. 3600 IN SRV 0 5 8140 puppet-server-a.example.com."),
				test.SRV("_x-puppet._tcp.example.com. 3600 IN SRV 0 5 8140 puppet-server-b.example.com."),
			},
			Extra: []dns.RR{
				test.AAAA("puppet-server-a.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:15"),
				test.AAAA("puppet-server-b.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:16"),
			},
		},
		{
			Qname: webdotexampledotcomName, Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				webdotexampledotcomRecordAAAA,
			},
		},
		{
			Qname: wwwdotexampledotcomName, Qtype: dns.TypeCNAME,
			Answer: []dns.RR{
				wwwdotexampledotcomRecordCNAME,
				webdotexampledotcomRecordAAAA,
			},
		},
		{
			Qname: wwwdotexampledotcomName, Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				wwwdotexampledotcomRecordCNAME,
				webdotexampledotcomRecordAAAA,
			},
		},
		{
			Qname: subdotexampledotcomName, Qtype: dns.TypeNS,
			Answer: subdotexampledotcomNS,
			Extra:  exampledotcomNSAddr6,
		},
		{
			Qname: "myservice.sub.example.com.", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				test.AAAA("myservice.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:10"),
			},
		},
		{
			Qname: subtwodotexampledotcomName, Qtype: dns.TypeNS,
			Answer: subtwodotexampledotcomNS,
			Extra:  exampledotcomNSAddr6,
		},
		{
			Qname: "myotherservice.subtwo.example.com.", Qtype: dns.TypeAAAA,
			Answer: []dns.RR{
				test.AAAA("myotherservice.subtwo.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:10"),
			},
		},
	}

	testLookupPTRV6 []test.Case = []test.Case{
		{
			Qname: "0.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("0.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR dns01.example.com."),
			},
		},
		{
			Qname: "1.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("1.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR dns02.example.com."),
			},
		},
		{
			Qname: "2.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("2.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR aservice.example.com."),
			},
		},
		{
			Qname: "3.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("3.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR mail.example.com."),
			},
		},
		{
			Qname: "5.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("5.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR puppet-server-a.example.com."),
			},
		},
		{
			Qname: "6.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("6.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR puppet-server-b.example.com."),
			},
		},
		{
			Qname: "7.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("7.1.0.0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR web.example.com."),
			},
		},
		{
			Qname: "0.1.0.0.2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("0.1.0.0.2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR myservice.sub.example.com."),
			},
		},
		{
			Qname: "0.1.0.0.3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa.", Qtype: dns.TypePTR,
			Answer: []dns.RR{
				test.PTR("0.1.0.0.3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN PTR myotherservice.subtwo.example.com."),
			},
		},
	}
)

func TestLookupRecords(t *testing.T) {
	RunTestLookup(t, testLookupRecordV4, testFamilyV4)
	RunTestLookup(t, testLookupPTRV4, testFamilyV4)
	RunTestLookup(t, testLookupRecordV6, testFamilyV6)
	RunTestLookup(t, testLookupPTRV6, testFamilyV6)
}

var (
	testUnknownRecords []test.Case = []test.Case{
		{
			Qname: "noop.com.", Qtype: dns.TypeSOA,
			Rcode: dns.RcodeNameError,
		},
	}

	testUnknownRecordsV4 []test.Case = []test.Case{
		{
			Qname: "noop.example.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeNameError,
		},
	}

	testUnknownRecordsV6 []test.Case = []test.Case{
		{
			Qname: "noop.example.com.", Qtype: dns.TypeAAAA,
			Rcode: dns.RcodeNameError,
		},
	}
)

func TestLookupUnknown(t *testing.T) {
	RunTestLookup(t, testUnknownRecords, testFamilyV4)
	RunTestLookup(t, testUnknownRecordsV4, testFamilyV4)
	RunTestLookup(t, testUnknownRecordsV6, testFamilyV6)
}

func TestOffline(t *testing.T) {
	netboxdns := NetboxDNS{
		Next:  test.ErrorHandler(),
		zones: []string{"."},
		requestClient: &netbox.APIRequestClient{
			Client: &http.Client{
				Timeout: defaultHTTPClientTimeout,
			},
			NetboxURL: &url.URL{
				Scheme: "http",
				Host:   "localhost:9876",
				Path:   testInstanceUrlPath,
			},
			Token: testInstanceToken,
		},
	}
	tc := test.Case{
		Qname: exampledotcomName, Qtype: dns.TypeA,
	}
	msg := tc.Msg()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	_, err := netboxdns.ServeDNS(context.Background(), rec, msg)
	if err == nil {
		t.Error("expected connection error, got none")
	}
}

func TestUnauthorized(t *testing.T) {
	netboxdns := NetboxDNS{
		Next:  test.ErrorHandler(),
		zones: []string{"."},
		requestClient: &netbox.APIRequestClient{
			Client: &http.Client{
				Timeout: defaultHTTPClientTimeout,
			},
			NetboxURL: &url.URL{
				Scheme: "http",
				Host:   testInstanceUrlHost,
				Path:   testInstanceUrlPath,
			},
			Token: "noop",
		},
	}
	tc := test.Case{
		Qname: exampledotcomName, Qtype: dns.TypeA,
	}
	msg := tc.Msg()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	_, err := netboxdns.ServeDNS(context.Background(), rec, msg)
	if err == nil {
		t.Error("expected connection error, got none")
	}
}

func TestFallthrough(t *testing.T) {
	netboxdns := NetboxDNS{
		Next:  test.ErrorHandler(),
		zones: []string{exampledotcomName},
		requestClient: &netbox.APIRequestClient{
			Client: &http.Client{
				Timeout: defaultHTTPClientTimeout,
			},
			NetboxURL: &url.URL{
				Scheme: "http",
				Host:   testInstanceUrlHost,
				Path:   testInstanceUrlPath,
			},
			Token: testInstanceToken,
		},
	}
	netboxdns.fall.SetZonesFromArgs([]string{"out.example.com"})
	tc := test.Case{
		Qname: "a.out.example.com.", Qtype: dns.TypeA,
	}
	msg := tc.Msg()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	_, err := netboxdns.ServeDNS(context.Background(), rec, msg)
	if err != nil {
		t.Errorf("expected fallthrough, got %v", err)
	}
}

func TestUnhandledZone(t *testing.T) {
	netboxdns := NetboxDNS{
		Next:  test.ErrorHandler(),
		zones: []string{exampledotcomName},
		requestClient: &netbox.APIRequestClient{
			Client: &http.Client{
				Timeout: defaultHTTPClientTimeout,
			},
			NetboxURL: &url.URL{
				Scheme: "http",
				Host:   testInstanceUrlHost,
				Path:   testInstanceUrlPath,
			},
			Token: testInstanceToken,
		},
	}
	tc := test.Case{
		Qname: "www.example.net.", Qtype: dns.TypeA,
	}
	msg := tc.Msg()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	_, err := netboxdns.ServeDNS(context.Background(), rec, msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
