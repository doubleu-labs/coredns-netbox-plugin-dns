package netboxdns

import (
	"strings"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
)

// getActiveZones returns active zones respecting the configured view filter.
// For the common single-view case the server-side ?view= parameter is used
// (efficient). For multi-view or view_exclude we fetch all active zones and
// filter client-side.
func (n *NetboxDNS) getActiveZones() ([]netbox.Zone, error) {
	if len(n.viewNames) <= 1 && len(n.viewExclude) == 0 {
		return netbox.GetZones(n.requestClient, []string{n.viewName})
	}
	zones, err := netbox.GetZones(n.requestClient, []string{""})
	if err != nil {
		return nil, err
	}
	return n.filterZonesByView(zones), nil
}

// getCatalogZones returns catalog zones respecting the configured view filter.
func (n *NetboxDNS) getCatalogZones() ([]netbox.Zone, error) {
	if len(n.viewNames) <= 1 && len(n.viewExclude) == 0 {
		return netbox.GetCatalogZones(n.requestClient, n.viewName)
	}
	zones, err := netbox.GetCatalogZones(n.requestClient, "")
	if err != nil {
		return nil, err
	}
	return n.filterZonesByView(zones), nil
}

// filterZonesByView applies the viewNames whitelist or viewExclude blacklist
// to a slice of zones. When no view filter is configured, all zones pass.
func (n *NetboxDNS) filterZonesByView(zones []netbox.Zone) []netbox.Zone {
	if len(n.viewNames) == 0 && len(n.viewExclude) == 0 {
		return zones
	}
	out := make([]netbox.Zone, 0, len(zones))
	for i := range zones {
		vn := zoneViewName(&zones[i])
		if len(n.viewNames) > 0 {
			if containsCI(n.viewNames, vn) {
				out = append(out, zones[i])
			}
		} else if len(n.viewExclude) > 0 {
			if !containsCI(n.viewExclude, vn) {
				out = append(out, zones[i])
			}
		}
	}
	return out
}

func zoneViewName(z *netbox.Zone) string {
	if z.View != nil {
		return z.View.Name
	}
	return ""
}

func containsCI(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
