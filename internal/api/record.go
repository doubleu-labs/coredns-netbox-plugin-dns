package api

import (
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

// Records is a slice of multiple Record.
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

func (rq *RecordQuery) encode() string {
	out := url.Values{}

	if rq.FQDN != "" {
		out.Set("fqdn", rq.FQDN)
	}

	if rq.Name != "" {
		out.Set("name", rq.Name)
	}

	if len(rq.Type) > 0 {
		for t := range slices.Values(rq.Type) {
			out.Add("type", t)
		}
	}

	if len(rq.TypeExclude) > 0 {
		for t := range slices.Values(rq.TypeExclude) {
			out.Add("type_exclude", t)
		}
	}

	if rq.Zone != nil {
		out.Set("zone_if", strconv.Itoa(rq.Zone.ID))
	}

	return out.Encode()
}

func (rq *RecordQuery) GetRecords(c *Client) (Records, error) {
	u := c.netboxURL.JoinPath("records", "/")
	u.RawQuery = rq.encode()
	rs, err := getMany[Record](c, u.String())
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
		zone, zoneErr := get[Zone](c, zoneURL.String())
		if zoneErr != nil {
			return rs, zoneErr
		}
		zoneTTLs[zone.ID] = zone.DefaultTTL
		rs[i].TTL = &zone.DefaultTTL
	}
	return rs, nil
}
