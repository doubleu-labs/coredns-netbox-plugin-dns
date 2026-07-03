package lookup

import (
	"fmt"
	"net/url"
	"slices"
	"sync"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

var (
	testClient     *api.Client
	testClientOnce sync.Once
)

func newTestClient(t *testing.T) *api.Client {
	t.Helper()
	testClientOnce.Do(
		func() {
			uri, token := testutil.GetTokenAndUrl(t)
			hostUrl, err := url.Parse(
				fmt.Sprintf(
					"%s/api/plugins/netbox-dns",
					uri,
				),
			)
			if err != nil {
				t.Fatalf("failed to parse url: %v", err)
			}
			testClient = api.NewClient(token, hostUrl)
		},
	)
	return testClient
}

func Test_Lookup(t *testing.T) {
	client := newTestClient(t)

	tests := []struct {
		QName string
		QType uint16
	}{
		{
			"example.com.",
			dns.TypeA,
		},
		{
			"invalid.com.",
			dns.TypeA,
		},
		{
			"example.com.",
			uint16(65444),
		},
		{
			"example.com.",
			dns.TypeSOA,
		},
		{
			"web.example.com.",
			dns.TypeA,
		},
		{
			"unknown.example.com.",
			dns.TypeA,
		},
		{
			"www.example.com.",
			dns.TypeA,
		},
		{
			"sub.example.com.",
			dns.TypeA,
		},
		{
			"example.com.",
			dns.TypeNS,
		},
		{
			"example.com.",
			dns.TypeMX,
		},
		{
			"_x-puppet._tcp.internal.example.com.",
			dns.TypeSRV,
		},
	}

	lookup := &Lookup{
		Client: client,
		Family: 1,
		Logger: new(log.NewWithPlugin("test-lookup")),
		QName:  "example.com.",
		QType:  dns.TypeA,
		Views:  &core.Views{},
	}

	for test := range slices.Values(tests) {
		name := fmt.Sprintf(
			"lookup_%s_%s",
			test.QName,
			dns.TypeToString[test.QType],
		)
		t.Run(
			name,
			func(t *testing.T) {
				lookup.QName = test.QName
				lookup.QType = test.QType
				resp, err := lookup.Run()
				if err != nil {
					t.Fatalf("lookup error: %v", err)
				}
				t.Logf("lookup response: %v", resp)
			},
		)
	}
}
