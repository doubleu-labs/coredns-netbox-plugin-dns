package api

import (
	"slices"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

type nameserver struct {
	Name string `json:"name"`
}

// Zone is a NetBox zone.
type Zone struct {
	DefaultTTL  uint32       `json:"default_ttl"`
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Nameservers []nameserver `json:"nameservers"`

	SOATTL     uint32     `json:"soa_ttl"`
	SOAMName   nameserver `json:"soa_mname"`
	SOARName   string     `json:"soa_rname"`
	SOASerial  uint32     `json:"soa_serial"`
	SOARefresh uint32     `json:"soa_refresh"`
	SOARetry   uint32     `json:"soa_retry"`
	SOAExpire  uint32     `json:"soa_expire"`
	SOAMinimum uint32     `json:"soa_minimum"`
}

// ZoneQuery represents parameters used for querying zones from the NetBox API.
type ZoneQuery struct {
	Views *core.Views
}

// GetZones returns a list of zones from the NetBox API matching the specified
// query parameters.
func (zq *ZoneQuery) GetZones(c *Client) ([]Zone, error) {
	u := c.netboxURL.JoinPath("zones", "/")
	q := u.Query()
	for status := range slices.Values(c.activeStatus) {
		q.Add("status", status)
	}
	for viewInclude := range slices.Values(zq.Views.Include) {
		q.Add("view", viewInclude)
	}
	for viewExclude := range slices.Values(zq.Views.Exclude) {
		q.Add("view__n", viewExclude)
	}
	u.RawQuery = q.Encode()
	zone, err := getMany[Zone](c, u.String())
	if err != nil {
		return nil, err
	}
	return zone, nil
}
