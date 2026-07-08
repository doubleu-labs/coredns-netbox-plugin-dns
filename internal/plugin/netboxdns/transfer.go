package netboxdns

import (
	"errors"

	"github.com/coredns/coredns/plugin/transfer"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/cache"
	"github.com/miekg/dns"
)

type zoneTransferCache interface {
	GetZoneNames() []string
	Put(*api.Zone, *dns.SOA, []dns.RR)
	Size() int
	Transfer(string, uint32) (<-chan []dns.RR, error)
}

// Transfer implements the transfer.Transfer interface.
func (n *netboxDNS) Transfer(zone string, serial uint32) (
	<-chan []dns.RR,
	error,
) {
	if n.zoneCache == nil {
		return nil, transfer.ErrNotAuthoritative
	}
	ch, err := n.zoneCache.Transfer(zone, serial)
	if err != nil {
		if errors.Is(err, cache.ErrZoneNotFound) {
			return nil, transfer.ErrNotAuthoritative
		}
		return nil, err
	}
	return ch, nil
}
