package mock

import (
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

type TransferCache struct {
	Ch        <-chan []dns.RR
	Err       error
	GotZone   string
	GotSerial uint32
	Called    bool
}

func (m *TransferCache) Transfer(
	zone string,
	serial uint32,
) (<-chan []dns.RR, error) {
	m.GotZone = zone
	m.GotSerial = serial
	m.Called = true
	return m.Ch, m.Err
}

func (m *TransferCache) GetZoneNames() []string {
	return nil
}

func (m *TransferCache) Put(_ *api.Zone, _ *dns.SOA, _ []dns.RR) {
	return
}

func (m *TransferCache) Size() int {
	return 0
}
