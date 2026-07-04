package netboxdns

import (
	"github.com/miekg/dns"
)

// Transfer implements the transfer.Transfer interface.
func (n *netboxDNS) Transfer(zone string, serial uint32) (
	<-chan []dns.RR,
	error,
) {
	if n.zoneCache == nil {
		return nil, nil
	}
	return n.zoneCache.Transfer(zone, serial)
}
