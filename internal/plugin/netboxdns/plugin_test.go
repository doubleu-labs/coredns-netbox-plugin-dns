package netboxdns

//noinspection LongLine
import (
	"context"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/test"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/plugin/netboxdns/poller"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func newDisabledViewPoller(t *testing.T) *poller.ViewPoller {
	t.Helper()

	vp, err := poller.NewViewPoller(
		testutil.NewServeDNSTestClient(t),
		&core.Views{
			Include: []string{"internal"},
		},
		time.Second,
	)
	if err != nil {
		t.Fatalf("new view poller error: %v", err)
	}
	vp.SetCanResolveForTest(false)
	return vp
}

func Test_NoOpFallthrough(t *testing.T) {
	next := &testutil.RecordingHandler{
		Rcode: dns.RcodeNameError,
	}
	n := &netboxDNS{
		Next:       next,
		client:     testutil.NewServeDNSTestClient(t),
		fall:       fall.Root,
		logger:     log.NewWithPlugin(pluginName),
		noop:       true,
		viewPoller: newDisabledViewPoller(t),
		views:      &core.Views{},
		zones:      []string{"."},
	}

	req := new(dns.Msg)
	req.SetQuestion("www.example.com.", dns.TypeA)
	rec := &testutil.RecordingResponseWriter{}

	rcode, err := n.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("servedns error: %v", err)
	}
	if rcode != dns.RcodeNameError {
		t.Fatalf("got rcode %d; want %d", rcode, dns.RcodeNameError)
	}
	if !next.Called() {
		t.Fatalf("next handler not called")
	}
}

func Test_NoOpNoFallthrough(t *testing.T) {
	next := &testutil.RecordingHandler{
		Rcode: dns.RcodeNameError,
	}
	n := &netboxDNS{
		Next:       next,
		client:     testutil.NewServeDNSTestClient(t),
		logger:     log.NewWithPlugin(pluginName),
		noop:       true,
		viewPoller: newDisabledViewPoller(t),
		views:      &core.Views{},
		zones:      []string{"."},
	}

	req := new(dns.Msg)
	req.SetQuestion("www.example.com.", dns.TypeA)
	rec := &testutil.RecordingResponseWriter{}

	rcode, err := n.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("servedns error: %v", err)
	}
	if rcode != dns.RcodeServerFailure {
		t.Fatalf("got rcode %d; want %d", rcode, dns.RcodeServerFailure)
	}
	if next.Called() {
		t.Fatalf("next handler called, want no fallthrough")
	}
}

func Test_FallthroughViewPollerDisabledAndFallthroughMatches(t *testing.T) {
	next := &testutil.RecordingHandler{
		Rcode: dns.RcodeNameError,
	}
	n := &netboxDNS{
		Next:       next,
		client:     testutil.NewServeDNSTestClient(t),
		fall:       fall.Root,
		logger:     log.NewWithPlugin(pluginName),
		viewPoller: newDisabledViewPoller(t),
		views: &core.Views{
			Include: []string{"internal"},
		},
		zones: []string{"example.com."},
	}

	req := new(dns.Msg).SetQuestion("www.example.com.", dns.TypeA)
	rec := &testutil.RecordingResponseWriter{}

	rcode, err := n.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("servedns error: %v", err)
	}
	if rcode != dns.RcodeNameError {
		t.Fatalf("got rcode %d; want %d", rcode, dns.RcodeNameError)
	}
	if !next.Called() {
		t.Fatalf("next handler not called")
	}
}

func Test_ReturnServfailWhenViewPollerDisabledAndNoFallthrough(t *testing.T) {
	next := &testutil.RecordingHandler{
		Rcode: dns.RcodeNameError,
	}
	n := &netboxDNS{
		Next:       next,
		client:     testutil.NewServeDNSTestClient(t),
		logger:     log.NewWithPlugin(pluginName),
		viewPoller: newDisabledViewPoller(t),
		views: &core.Views{
			Include: []string{"internal"},
		},
		zones: []string{"example.com."},
	}

	req := new(dns.Msg).SetQuestion("www.example.com.", dns.TypeA)
	rec := &testutil.RecordingResponseWriter{}

	rcode, err := n.ServeDNS(context.Background(), rec, req)
	if err != nil {
		t.Fatalf("servedns error: %v", err)
	}
	if rcode != dns.RcodeServerFailure {
		t.Fatalf("got rcode %d; want %d", rcode, dns.RcodeServerFailure)
	}
	if next.Called() {
		t.Fatalf("next handler called, want no fallthrough")
	}
}

func Test_Lookup(t *testing.T) {
	//noinspection LongLine
	tests := []testutil.LookupTestCase{
		{
			Case: test.Case{
				Qname: "www.example.com.",
				Qtype: dns.TypeA,
				Answer: []dns.RR{
					test.CNAME("www.example.com. 3600 IN CNAME web.example.com."),
					test.A("web.example.com. 3600 IN A 10.0.1.17"),
					test.AAAA("web.example.com. 3600 IN AAAA 2001:db8:dead:beef::1:17"),
				},
			},
		},
		{
			Case: test.Case{
				Qname: "example.com.",
				Qtype: dns.TypeSOA,
				Answer: []dns.RR{
					test.SOA("example.com. 86400 IN SOA ns1.example.com. hostmaster.example.com. 2023010100 7200 1800 604800 3600"),
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
				Qname: "invalid.com.",
				Qtype: dns.TypeA,
			},
			WantErr: true,
			Rcode:   dns.RcodeNameError,
		},
		{
			Case: test.Case{
				Qname: "sub.example.com.",
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
		token <!TOKEN>
		url <!URL>
	}`
	server := testutil.NewTestServer(t, corefile)
	defer server.Close(t)

	testutil.RunLookupTests(t, tests, server)
}
