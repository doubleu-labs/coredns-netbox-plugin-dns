package netbox

import (
	"net/url"
	"strconv"
	"strings"
)

type Zone struct {
	DefaultTTL  uint32     `json:"default_ttl"`
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	NameServers []SOAMName `json:"nameservers"`
	View        *View      `json:"view"`

	// SOA fields populated by netbox-dns. soa_mname is a nested nameserver
	// object (same shape as entries in NameServers).
	SOATTL     uint32   `json:"soa_ttl"`
	SOAMName   SOAMName `json:"soa_mname"`
	SOARName   string   `json:"soa_rname"`
	SOASerial  uint32   `json:"soa_serial"`
	SOARefresh uint32   `json:"soa_refresh"`
	SOARetry   uint32   `json:"soa_retry"`
	SOAExpire  uint32   `json:"soa_expire"`
	SOAMinimum uint32   `json:"soa_minimum"`
}

type SOAMName struct {
	Name string `json:"name"`
}

// View represents a netbox-dns view. Only the fields the plugin actually
// consumes are decoded; the upstream API returns more.
type View struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func urlZones(netboxurl *url.URL) *url.URL {
	return netboxurl.JoinPath("zones", "/")
}

func urlZoneID(netboxurl *url.URL, id int) *url.URL {
	return netboxurl.JoinPath("zones", "/", strconv.Itoa(id), "/")
}

// GetCatalogZones returns zones tagged for catalog publication. The
// convention used by the plugin is "status=parked AND name has prefix
// 'cat.'": parked alone could be a real zone temporarily out of service,
// and the cat. prefix alone could collide with a real domain, but the
// AND of both is an unambiguous opt-in declaration. The viewName scoping
// behaves the same as for GetZones (per-view catalog zones are supported).
//
// Name-prefix filtering is done client-side because NetBox-dns has no
// "zone name starts with" API filter.
func GetCatalogZones(requestClient *APIRequestClient, viewName string) ([]Zone, error) {
	requestUrl := urlZones(requestClient.NetboxURL)
	q := requestUrl.Query()
	q.Set("status", "parked")
	if viewName != "" {
		q.Set("view", viewName)
	}
	requestUrl.RawQuery = q.Encode()
	all, err := getMany[Zone](requestClient, requestUrl.String())
	if err != nil {
		return nil, err
	}
	out := make([]Zone, 0, len(all))
	for i := range all {
		if strings.HasPrefix(strings.ToLower(all[i].Name), "cat.") {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// GetZones returns the active zones managed by netbox-dns. When viewName
// is non-empty the result is filtered server-side to that view (NetBox
// supports `?view=<name>` on /api/plugins/netbox-dns/zones/).
//
// Only zones with status=active are returned. NetBox-dns also has parked,
// deprecated, and reserved statuses; those represent zones that exist as
// records-of-record but should not be served as authoritative DNS, so we
// exclude them from every normal serving path (lookup, AXFR, IXFR poller).
func GetZones(requestClient *APIRequestClient, viewName string) ([]Zone, error) {
	requestUrl := urlZones(requestClient.NetboxURL)
	q := requestUrl.Query()
	q.Set("status", "active")
	if viewName != "" {
		q.Set("view", viewName)
	}
	requestUrl.RawQuery = q.Encode()
	zones, err := getMany[Zone](requestClient, requestUrl.String())
	if err != nil {
		return nil, err
	}
	return zones, nil
}
