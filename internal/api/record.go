package api

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

var txtMultiValueRegexp *regexp.Regexp

func init() {
	txtMultiValueRegexp = regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
}

type Record struct {
	AbsoluteValue string  `json:"absolute_value"`
	FQDN          string  `json:"fqdn"`
	TTL           *uint32 `json:"ttl"`
	Type          string  `json:"type"`
	Zone          Zone    `json:"zone"`
}

func (r *Record) ToRR() (dns.RR, error) {
	qtype := dns.StringToType[r.Type]
	if qtype == dns.TypeTXT {
		var txtValues []string
		if strings.HasPrefix(r.AbsoluteValue, `"`) {
			values := txtMultiValueRegexp.FindAllString(r.AbsoluteValue, -1)
			for i := range values {
				values[i] = strings.Trim(values[i], `"`)
				values[i] = strings.ReplaceAll(values[i], "\\r\\n", "")
				values[i] = strings.ReplaceAll(values[i], "\\n", "")
				values[i] = strings.TrimSpace(values[i])
				if values[i] != "" {
					txtValues = append(txtValues, values[i])
				}
			}
		} else {
			txtValues = append(txtValues, r.AbsoluteValue)
		}
		rr := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   r.FQDN,
				Ttl:    *r.TTL,
				Class:  dns.ClassINET,
				Rrtype: dns.TypeTXT,
			},
			Txt: txtValues,
		}
		return rr, nil
	}
	recordString := fmt.Sprintf(
		"%s %d IN %s %s",
		r.FQDN,
		*r.TTL,
		dns.TypeToString[qtype],
		r.AbsoluteValue,
	)
	rr, err := dns.NewRR(recordString)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

// Records is a slice of Record used for methods that return multiple Record
//
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Records []Record

func (rs *Records) ToRRs() ([]dns.RR, error) {
	rrs := make([]dns.RR, 0, len(*rs))
	for r := range slices.Values(*rs) {
		rr, err := r.ToRR()
		if err != nil {
			return rrs, err
		}
		rrs = append(rrs, rr)
	}
	return rrs, nil
}

type RecordQuery struct {
	FQDN        string
	Name        string
	Type        []string
	TypeExclude []string
	Zone        *Zone
}

func (rq *RecordQuery) encode(u *url.URL) string {
	q := u.Query()

	if rq.FQDN != "" {
		q.Set("fqdn", rq.FQDN)
	}

	if rq.Name != "" {
		q.Set("name", rq.Name)
	}

	if len(rq.Type) > 0 {
		for t := range slices.Values(rq.Type) {
			q.Add("type", t)
		}
	}

	if len(rq.TypeExclude) > 0 {
		for t := range slices.Values(rq.TypeExclude) {
			q.Add("type__n", t)
		}
	}

	if rq.Zone != nil {
		q.Set("zone_id", strconv.Itoa(rq.Zone.ID))
	}

	return q.Encode()
}

func (rq *RecordQuery) GetRecords(ctx context.Context, c *Client) (
	Records,
	error,
) {
	u := c.netboxURL.JoinPath("records", "/")
	u.RawQuery = rq.encode(u)
	rs, err := getMany[Record](ctx, c, u.String())
	if err != nil {
		return nil, err
	}
	if rq.Zone != nil {
		for i, r := range rs {
			if r.TTL == nil {
				rs[i].TTL = &rq.Zone.DefaultTTL
			}
		}
		return rs, nil
	}
	zoneTTLs := make(map[int]uint32)
	for i, r := range rs {
		if r.TTL != nil {
			continue
		}
		if ttl, ok := zoneTTLs[r.Zone.ID]; ok {
			rs[i].TTL = &ttl
			continue
		}
		zoneURL := c.netboxURL.JoinPath("zones", strconv.Itoa(r.Zone.ID), "/")
		var zoneErr error
		zone, zoneErr := get[Zone](ctx, c, zoneURL.String())
		if zoneErr != nil {
			return rs, zoneErr
		}
		zoneTTLs[zone.ID] = zone.DefaultTTL
		rs[i].TTL = &zone.DefaultTTL
	}
	return rs, nil
}
