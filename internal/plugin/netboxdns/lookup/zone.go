package lookup

import (
	"context"
	"slices"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

func (l *Lookup) matchZone(n string) (*api.Zone, error) {
	zoneQuery := &api.ZoneQuery{
		ActiveZoneStatus: l.ActiveZoneStatus,
		Views:            l.Views,
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()
	zones, err := zoneQuery.GetZones(ctx, l.Client)
	if err != nil {
		return nil, err
	}
	var out *api.Zone
	for zone := range slices.Values(zones) {
		if dns.IsSubDomain(
			zone.Name,
			n,
		) && (out == nil || len(zone.Name) > len(out.Name)) {
			out = &zone
		}
	}
	return out, nil
}
