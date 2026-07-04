package cache

import (
	"strings"

	"github.com/miekg/dns"
)

func normalizeZoneName(zone string) string {
	return dns.Fqdn(strings.ToLower(zone))
}
