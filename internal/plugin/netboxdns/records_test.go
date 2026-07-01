package netboxdns

import (
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func Test_Records(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Name: "unknown record qtype",
			Case: test.Case{
				Qname: "example.com",
				Qtype: 65534,
				Ns: []dns.RR{
					test.SOA("example.com. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
			},
		},
		{
			Name: "unknown record qtype unknown zone",
			Case: test.Case{
				Qname: "invalid.com",
				Qtype: 65534,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
		{
			Case: test.Case{
				Qname: "noop.invalid.com",
				Qtype: dns.TypeSOA,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_SOARecords(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA ns1.sub.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns1.sub.example.com."),
					test.NS("0.1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns2.sub.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.sub.example.com. 3600 IN A 10.0.10.10"),
					test.AAAA("ns1.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::10:10"),
					test.A("ns2.sub.example.com. 3600 IN A 10.0.10.11"),
					test.AAAA("ns2.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::10:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns1.example.com."),
					test.NS("1.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns2.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.example.com. 3600 IN A 10.0.1.10"),
					test.AAAA("ns1.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:10"),
					test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
					test.AAAA("ns2.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "1.0.10.in-addr.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("1.0.10.in-addr.arpa. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("1.0.10.in-addr.arpa. 3600 IN NS ns1.example.com."),
					test.NS("1.0.10.in-addr.arpa. 3600 IN NS ns2.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.example.com. 3600 IN A 10.0.1.10"),
					test.AAAA("ns1.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:10"),
					test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
					test.AAAA("ns2.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA ns1.acme.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns1.acme.example.com."),
					test.NS("2.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns2.acme.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.acme.example.com. 3600 IN A 10.0.2.10"),
					test.AAAA("ns1.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:10"),
					test.A("ns2.acme.example.com. 3600 IN A 10.0.2.11"),
					test.AAAA("ns2.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "2.0.10.in-addr.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("2.0.10.in-addr.arpa. 86400 IN SOA ns1.acme.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("2.0.10.in-addr.arpa. 3600 IN NS ns1.acme.example.com."),
					test.NS("2.0.10.in-addr.arpa. 3600 IN NS ns2.acme.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.acme.example.com. 3600 IN A 10.0.2.10"),
					test.AAAA("ns1.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:10"),
					test.A("ns2.acme.example.com. 3600 IN A 10.0.2.11"),
					test.AAAA("ns2.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 86400 IN SOA ns1.internal.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns1.internal.example.com."),
					test.NS("3.0.0.0.0.0.0.0.0.0.0.0.f.e.e.b.d.a.e.d.8.b.d.0.1.0.0.2.ip6.arpa. 3600 IN NS ns2.internal.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.internal.example.com. 3600 IN A 10.0.3.10"),
					test.AAAA("ns1.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:10"),
					test.A("ns2.internal.example.com. 3600 IN A 10.0.3.11"),
					test.AAAA("ns2.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "3.0.10.in-addr.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("3.0.10.in-addr.arpa. 86400 IN SOA ns1.internal.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("3.0.10.in-addr.arpa. 3600 IN NS ns1.internal.example.com."),
					test.NS("3.0.10.in-addr.arpa. 3600 IN NS ns2.internal.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.internal.example.com. 3600 IN A 10.0.3.10"),
					test.AAAA("ns1.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:10"),
					test.A("ns2.internal.example.com. 3600 IN A 10.0.3.11"),
					test.AAAA("ns2.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "10.0.10.in-addr.arpa",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("10.0.10.in-addr.arpa. 86400 IN SOA ns1.sub.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("10.0.10.in-addr.arpa. 3600 IN NS ns1.sub.example.com."),
					test.NS("10.0.10.in-addr.arpa. 3600 IN NS ns2.sub.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.sub.example.com. 3600 IN A 10.0.10.10"),
					test.AAAA("ns1.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::10:10"),
					test.A("ns2.sub.example.com. 3600 IN A 10.0.10.11"),
					test.AAAA("ns2.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::10:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "acme.example.com",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("acme.example.com. 86400 IN SOA ns1.acme.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("acme.example.com. 3600 IN NS ns1.acme.example.com."),
					test.NS("acme.example.com. 3600 IN NS ns2.acme.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.acme.example.com. 3600 IN A 10.0.2.10"),
					test.AAAA("ns1.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:10"),
					test.A("ns2.acme.example.com. 3600 IN A 10.0.2.11"),
					test.AAAA("ns2.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "example.com",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("example.com. 86400 IN SOA ns1.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("example.com. 3600 IN NS ns1.example.com."),
					test.NS("example.com. 3600 IN NS ns2.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.example.com. 3600 IN A 10.0.1.10"),
					test.AAAA("ns1.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:10"),
					test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
					test.AAAA("ns2.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "internal.example.com",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("internal.example.com. 86400 IN SOA ns1.internal.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("internal.example.com. 3600 IN NS ns1.internal.example.com."),
					test.NS("internal.example.com. 3600 IN NS ns2.internal.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.internal.example.com. 3600 IN A 10.0.3.10"),
					test.AAAA("ns1.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:10"),
					test.A("ns2.internal.example.com. 3600 IN A 10.0.3.11"),
					test.AAAA("ns2.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "sub.example.com",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("sub.example.com. 86400 IN SOA ns1.sub.example.com. admin.example.com. 1 43200 7200 2419200 3600"),
				},
				Ns: []dns.RR{
					test.NS("sub.example.com. 3600 IN NS ns1.sub.example.com."),
					test.NS("sub.example.com. 3600 IN NS ns2.sub.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.sub.example.com. 3600 IN A 10.0.10.10"),
					test.AAAA("ns1.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::10:10"),
					test.A("ns2.sub.example.com. 3600 IN A 10.0.10.11"),
					test.AAAA("ns2.sub.example.com. 3600 IN AAAA 2001:db8:dead:beef::10:11"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_NSRecords(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "example.com",
				Qtype: dns.TypeNS,
				Answer: []dns.RR{
					test.NS("example.com. 3600 IN NS ns1.example.com."),
					test.NS("example.com. 3600 IN NS ns2.example.com."),
				},
				Extra: []dns.RR{
					test.A("ns1.example.com. 3600 IN A 10.0.1.10"),
					test.AAAA("ns1.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:10"),
					test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
					test.AAAA("ns2.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:11"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_CNAMERecords(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "www.example.com",
				Qtype: dns.TypeCNAME,
				Answer: []dns.RR{
					test.CNAME("www.example.com. 3600 IN CNAME web.example.com."),
					test.A("web.example.com. 3600 IN A 10.0.1.17"),
					test.AAAA("web.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:17"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_MXRecords(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "example.com",
				Qtype: dns.TypeMX,
				Answer: []dns.RR{
					test.MX("example.com. 3600 IN MX 10 mail.example.com."),
				},
				Extra: []dns.RR{
					test.A("mail.example.com. 3600 IN A 10.0.1.13"),
					test.AAAA("mail.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:13"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_TXTRecords(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "example.com",
				Qtype: dns.TypeTXT,
				Answer: []dns.RR{
					test.TXT(`example.com. 3600 IN TXT "my value" "second my value" "third my value"`),
					test.TXT(`example.com. 3600 IN TXT "newline record" "second value"`),
					test.TXT(`example.com. 3600 IN TXT "some value" "another value"`),
					test.TXT(`example.com. 3600 IN TXT "v=DMARC1;p=none;sp=quarantine;pct=100;rua=admin@example.com;"`),
					test.TXT(`example.com. 3600 IN TXT "v=spf1 ip4:10.0.0.13 ip6:2001:db8:dead:beef::1:13 a -all"`),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_SRVRecords(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "_x-puppet._tcp.internal.example.com",
				Qtype: dns.TypeSRV,
				Answer: []dns.RR{
					test.SRV("_x-puppet._tcp.internal.example.com. 3600 IN SRV 0 5 8140 puppet1.internal.example.com."),
					test.SRV("_x-puppet._tcp.internal.example.com. 3600 IN SRV 0 5 8140 puppet2.internal.example.com."),
				},
				Extra: []dns.RR{
					test.A("puppet1.internal.example.com. 3600 IN A 10.0.3.15"),
					test.AAAA("puppet1.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:15"),
					test.A("puppet2.internal.example.com. 3600 IN A 10.0.3.16"),
					test.AAAA("puppet2.internal.example.com. 3600 IN AAAA 2001:db8:dead:beef::3:16"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}
