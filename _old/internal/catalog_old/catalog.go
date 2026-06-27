package catalog_old

import (
	"slices"
	"strings"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

const Version = "2"

func buildCatalog(c *netbox.Zone, m []netbox.Zone) []dns.RR {
	out := make([]dns.RR, 0, 2+len(m))

	return out
}

func catalogMemberCount(c *netbox.Zone, a []netbox.Zone) int {
	cl := strings.ToLower(strings.TrimSuffix(c.Name, "."))
	n := 0
	for v := range slices.Values(a) {
		if strings.EqualFold(v.Name, cl) {
			continue
		}
		n++
	}
	return n
}
