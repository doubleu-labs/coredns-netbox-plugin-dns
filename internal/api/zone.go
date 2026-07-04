package api

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/miekg/dns"
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

func (z *Zone) ToRR() ([]dns.RR, error) {
	soaStr := fmt.Sprintf(
		"%s. %d IN SOA %s. %s. %d %d %d %d %d",
		z.Name,
		z.SOATTL,
		z.SOAMName.Name,
		z.SOARName,
		z.SOASerial,
		z.SOARefresh,
		z.SOARetry,
		z.SOAExpire,
		z.SOAMinimum,
	)
	soaRR, err := dns.NewRR(soaStr)
	if err != nil {
		return nil, err
	}
	return []dns.RR{soaRR}, nil
}

// ZoneQuery represents parameters used for querying zones from the NetBox API.
type ZoneQuery struct {
	ActiveZoneStatus []string
	Views            *core.Views
}

func (zq *ZoneQuery) encode(_ *Client, u *url.URL) string {
	q := u.Query()

	for status := range slices.Values(zq.ActiveZoneStatus) {
		q.Add("status", status)
	}

	for vi := range slices.Values(zq.Views.Include) {
		q.Add("view", vi)
	}

	for ve := range slices.Values(zq.Views.Exclude) {
		q.Add("view__n", ve)
	}

	return q.Encode()
}

// GetZones returns a list of zones from the NetBox API matching the specified
// query parameters.
func (zq *ZoneQuery) GetZones(ctx context.Context, c *Client) ([]Zone, error) {
	u := c.netboxURL.JoinPath("zones", "/")
	u.RawQuery = zq.encode(c, u)
	zone, err := getMany[Zone](ctx, c, u.String())
	if err != nil {
		return nil, err
	}
	return zone, nil
}
