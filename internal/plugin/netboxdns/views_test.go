package netboxdns

import (
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func Test_ViewsExplicit(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "example.com",
				Qtype: dns.TypeA,
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
		{
			Case: test.Case{
				Qname: "ns1.internal.example.com",
				Qtype: dns.TypeA,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
		{
			Case: test.Case{
				Qname: "ns1.acme.example.com",
				Qtype: dns.TypeAAAA,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
		views public
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_ViewsExclude(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "ns1.internal.example.com",
				Qtype: dns.TypeA,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
		{
			Case: test.Case{
				Qname: "ns1.acme.example.com",
				Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("ns1.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:10"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "ns2.example.com",
				Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
		views_exclude internal
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}

func Test_ViewsImplicitExclude(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "ns1.internal.example.com",
				Qtype: dns.TypeA,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
		{
			Case: test.Case{
				Qname: "ns2.example.com",
				Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.A("ns2.example.com. 3600 IN A 10.0.1.11"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "ns1.acme.example.com",
				Qtype: dns.TypeAAAA,
				Answer: []dns.RR{
					test.AAAA("ns1.acme.example.com. 3600 IN AAAA 2001:db8:dead:beef::2:10"),
				},
			},
		},
	}

	corefile := `netboxdns {
		url <!URL>
		token <!TOKEN>
		views public offsite
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}
