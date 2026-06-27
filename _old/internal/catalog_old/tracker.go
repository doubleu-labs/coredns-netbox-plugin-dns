package catalog_old

import (
	"sync"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
)

type Tracker struct {
	mu        sync.Mutex
	serials   map[string]uint32
	prevPrint map[string]string
}

func NewTracker() *Tracker {
	return &Tracker{
		serials:   make(map[string]uint32),
		prevPrint: make(map[string]string),
	}
}

func (t *Tracker) NextSerial(c string, m []netbox.Zone) uint32 {
	// TODO: implement
	return 0
}
