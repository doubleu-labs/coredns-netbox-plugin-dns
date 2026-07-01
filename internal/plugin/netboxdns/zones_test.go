package netboxdns

import (
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func Test_LookupDelegate(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "sub.example.com",
				Qtype: dns.TypeA,
				Answer: []dns.RR{
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
