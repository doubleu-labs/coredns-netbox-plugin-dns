package netboxdns

import (
	"errors"
	"testing"

	"github.com/coredns/coredns/plugin/transfer"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/cache"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func TestMockTransfer_AXFRDelegatesToZoneCache(t *testing.T) {
	wantCh := make(chan []dns.RR)
	close(wantCh)

	mockCache := &testutil.MockTransferCache{
		Ch: wantCh,
	}
	n := &netboxDNS{
		zoneCache: mockCache,
	}

	gotCh, err := n.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("transfer error: want nil, got %v", err)
	}
	if !mockCache.Called {
		t.Fatalf("zone cache not called")
	}
	if mockCache.GotZone != "example.com." {
		t.Fatalf("zone: got %q; want %q", mockCache.GotZone, "example.com.")
	}
	if mockCache.GotSerial != 0 {
		t.Fatalf("serial: got %d; want %d", mockCache.GotSerial, 0)
	}
	if gotCh != wantCh {
		t.Fatalf("returned channel does not match delegated channel")
	}
}

func TestMockTransfer_IXFRDelegatesToZoneCache(t *testing.T) {
	wantCh := make(chan []dns.RR)
	close(wantCh)

	mockCache := &testutil.MockTransferCache{
		Ch: wantCh,
	}
	n := &netboxDNS{
		zoneCache: mockCache,
	}

	gotCh, err := n.Transfer("example.com.", 2026010100)
	if err != nil {
		t.Fatalf("transfer error: want nil, got %v", err)
	}
	if !mockCache.Called {
		t.Fatalf("zone cache not called")
	}
	if mockCache.GotZone != "example.com." {
		t.Fatalf("zone: got %q; want %q", mockCache.GotZone, "example.com.")
	}
	if mockCache.GotSerial != 2026010100 {
		t.Fatalf("serial: got %d; want %d", mockCache.GotSerial, 2026010100)
	}
	if gotCh != wantCh {
		t.Fatalf("returned channel does not match delegated channel")
	}
}

func TestMockTransfer_NilCacheIsNotAuthoritative(t *testing.T) {
	n := &netboxDNS{}

	ch, err := n.Transfer("example.com.", 0)
	if !errors.Is(err, transfer.ErrNotAuthoritative) {
		t.Fatalf(
			"transfer error: want %v, got %v",
			transfer.ErrNotAuthoritative,
			err,
		)
	}
	if ch != nil {
		t.Fatalf("channel: want nil, got %v", ch)
	}
}

func TestMockTransfer_ZoneNotFoundIsNotAuthoritative(t *testing.T) {
	mockCache := &testutil.MockTransferCache{
		Err: cache.ErrZoneNotFound,
	}
	n := &netboxDNS{
		zoneCache: mockCache,
	}

	ch, err := n.Transfer("missing.example.", 0)
	if !errors.Is(err, transfer.ErrNotAuthoritative) {
		t.Fatalf(
			"transfer error: want %v, got %v",
			transfer.ErrNotAuthoritative,
			err,
		)
	}
	if ch != nil {
		t.Fatalf("channel: want nil, got %v", ch)
	}
}

func TestMockTransfer_PropagateUnexpectedCacheError(t *testing.T) {
	wantErr := errors.New("cache failed")
	mockCache := &testutil.MockTransferCache{
		Err: wantErr,
	}
	n := &netboxDNS{
		zoneCache: mockCache,
	}

	ch, err := n.Transfer("example.com.", 0)
	if !errors.Is(err, wantErr) {
		t.Fatalf("transfer error: want %v, got %v", wantErr, err)
	}
	if ch != nil {
		t.Fatalf("channel: want nil, got %v", ch)
	}
}
