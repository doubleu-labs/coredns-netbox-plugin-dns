package lookup

import (
	"strings"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

// matchZone returns the zone that matches the given name. If the name exists in
// multiple views, the last checked view is returned and a warning is logged.
func (l *Lookup) matchZone(n string) (*netbox.Zone, error) {
	zs, err := netbox.GetZones(l.Client, l.settledViews)
	if err != nil {
		return nil, err
	}
	var out *netbox.Zone
	var amb int
	for i := range zs {
		z := &zs[i]
		if dns.IsSubDomain(z.Name, n) {
			if out == nil {
				out = z
			}
			if len(z.Name) > len(out.Name) {
				out = z
			}
			if strings.EqualFold(z.Name, n) {
				amb++
			}
		}
	}
	if amb > 1 && len(l.settledViews) == 0 {
		l.Logger.Warningf(
			"zone %q exists in %d views; configure 'view' or 'view_exclude' "+
				"to disambiguate",
			n,
			amb,
		)
	}
	return out, nil
}
