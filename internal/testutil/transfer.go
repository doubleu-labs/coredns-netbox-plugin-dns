package testutil

import (
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

type MockTransferCache struct {
	Ch        <-chan []dns.RR
	Err       error
	GotZone   string
	GotSerial uint32
	Called    bool
}

func (m *MockTransferCache) Transfer(
	zone string,
	serial uint32,
) (<-chan []dns.RR, error) {
	m.GotZone = zone
	m.GotSerial = serial
	m.Called = true
	return m.Ch, m.Err
}

func (m *MockTransferCache) GetZoneNames() []string {
	return nil
}

func (m *MockTransferCache) Put(_ *api.Zone, _ *dns.SOA, _ []dns.RR) {
	return
}

func (m *MockTransferCache) Size() int {
	return 0
}
