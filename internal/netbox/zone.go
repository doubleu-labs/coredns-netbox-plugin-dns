package netbox

import (
	"net/url"
	"strconv"
)

// Zone represents a DNS zone provided by the Netbox API. Only the fields the
// plugin actually consumes are decoded.
type Zone struct {
	DefaultTTL  uint32       `json:"default_ttl"`
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Nameservers []Nameserver `json:"nameservers"`
	View        *View        `json:"view"`

	// SOA fields populated by netbox-dns. soa_mname is a nested nameserver
	// object (same shape as entries in NameServers).
	SOATTL     uint32     `json:"soa_ttl"`
	SOAMName   Nameserver `json:"soa_mname"`
	SOARName   string     `json:"soa_rname"`
	SOASerial  uint32     `json:"soa_serial"`
	SOARefresh uint32     `json:"soa_refresh"`
	SOARetry   uint32     `json:"soa_retry"`
	SOAExpire  uint32     `json:"soa_expire"`
	SOAMinimum uint32     `json:"soa_minimum"`
}

// Nameserver represents a nameserver provided by the Netbox API.
type Nameserver struct {
	Name string `json:"name"`
}

// View represents a netbox-dns view. Only the fields the plugin actually
// consumes are decoded; the upstream API returns more.
type View struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func urlZones(u *url.URL) *url.URL {
	return u.JoinPath("zones", "/")
}

func urlZoneID(u *url.URL, id int) *url.URL {
	return u.JoinPath("zones", "/", strconv.Itoa(id), "/")
}

// GetCatalogZones returns zones tagged for catalog publication. The
// convention used by the plugin is "status=parked AND name has the prefix
// 'cat.'": parked alone could be a real zone temporarily out of service,
// and the cat. Prefix alone could collide with a real domain, but the
// AND of both is an unambiguous opt-in declaration. The viewName scoping
// behaves the same as for GetZones (per-view catalog zones are supported).
//
// Name-prefix filtering is done client-side because NetBox-dns has no
// "zone name starts with" API filter.
func GetCatalogZones(c *Client, v []string, p string) ([]Zone, error) {
	u := urlZones(c.NetboxURL)
	q := u.Query()
	q.Set("status", "parked")
	for _, n := range v {
		q.Add("view", n)
	}
	q.Set("name__isw", p)
	u.RawQuery = q.Encode()
	zs, err := getMany[Zone](c, u.String())
	if err != nil {
		return nil, err
	}
	return zs, nil
}

// GetZones returns the active zones managed by netbox-dns. When viewName
// is non-empty, the result is filtered server-side to that view (NetBox
// supports `?view=<name>` on /api/plugins/netbox-dns/zones/).
//
// Only zones with status=active are returned. NetBox-dns also have parked,
// deprecated, and reserved statuses; those represent zones that exist as
// records-of-record but should not be served as authoritative DNS, so we
// exclude them from every normal serving path (lookup, AXFR, IXFR poller).
func GetZones(c *Client, v []string) ([]Zone, error) {
	u := urlZones(c.NetboxURL)
	q := u.Query()
	q.Set("status", "active")
	for _, n := range v {
		q.Add("view", n)
	}
	u.RawQuery = q.Encode()
	z, err := getMany[Zone](c, u.String())
	if err != nil {
		return nil, err
	}
	return z, nil
}
