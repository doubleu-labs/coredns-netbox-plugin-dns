package catalog

import "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"

type Tracker struct{}

func (t *Tracker) NextSerial(c string, zs []netbox.Zone) uint32 {
	// TODO: implement
	return 0
}
