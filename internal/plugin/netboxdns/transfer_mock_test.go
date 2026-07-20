package netboxdns

import (
	"errors"
	"slices"
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/plugin/transfer"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/cache"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil/mock"
	"github.com/miekg/dns"
)

func TestMockTransfer_AXFRDelegatesToZoneCache(t *testing.T) {
	wantCh := make(chan []dns.RR)
	close(wantCh)

	mockCache := &mock.TransferCache{
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

	mockCache := &mock.TransferCache{
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
	mockCache := &mock.TransferCache{
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
	mockCache := &mock.TransferCache{
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

func TestMockTransfer_IXFR(t *testing.T) {
	n := &netboxDNS{
		zoneCache: cache.NewCache(4),
	}

	// noinspection LongLine
	soa90 := test.SOA(
		"example.com. 86400 IN SOA ns1.example.com. admin.example.com. 90 43200 7200 2419200 3600",
	)
	zone90 := &api.Zone{
		Name:      "example.com.",
		SOASerial: 90,
	}
	rrs90 := []dns.RR{
		test.A("www.example.com. 3600 IN A 192.168.2.10"),
	}
	n.zoneCache.Put(zone90, soa90, rrs90)

	// noinspection LongLine
	soa91 := test.SOA(
		"example.com. 86400 IN SOA ns1.example.com. admin.example.com. 91 43200 7200 2419200 3600",
	)
	zone91 := &api.Zone{
		Name:      "example.com.",
		SOASerial: 91,
	}
	rrs91 := []dns.RR{
		test.A("example.com. 3600 IN A 192.168.2.11"),
	}
	n.zoneCache.Put(zone91, soa91, rrs91)

	ch, err := n.Transfer("example.com.", 90)
	if err != nil {
		t.Fatalf("transfer error: want nil, got %v", err)
	}

	var got []dns.RR
	for batch := range ch {
		got = slices.Concat(got, batch)
	}
}
