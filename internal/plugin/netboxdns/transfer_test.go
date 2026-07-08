package netboxdns

import (
	"slices"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func TestTransfer_AXFR(t *testing.T) {
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
	msg.SetQuestion("example.com.", dns.TypeAXFR)

	resp, err := server.Transfer(msg)
	if err != nil {
		t.Fatal(err)
	}

	for envelope := range resp {
		if envelope.Error != nil {
			t.Fatalf("error during transfer: %v", envelope.Error)
		}
		for rr := range slices.Values(envelope.RR) {
			t.Logf("AXFR response: %v", rr)
		}
	}
}
