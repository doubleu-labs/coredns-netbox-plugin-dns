package netbox

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

var txtMultiValueRegexp *regexp.Regexp

func init() {
	txtMultiValueRegexp = regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
}

type Record struct {
	Type          string  `json:"type"`
	AbsoluteValue string  `json:"absolute_value"`
	TTL           *uint32 `json:"ttl"`
	Zone          Zone    `json:"zone"`
	FQDN          string  `json:"fqdn"`
}

type Records []Record

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

	if len(rq.Type) != 0 {
		for _, t := range rq.Type {
			out.Add("type", t)
		}
	}

	if len(rq.TypeExclude) != 0 {
		for _, t := range rq.TypeExclude {
			out.Add("type__n", t)
		}
	}

	if rq.Zone != nil {
		out.Set("zone_id", strconv.Itoa(rq.Zone.ID))
	}

	return out.Encode()
}

func (rq *RecordQuery) GetRecords(c *Client) (Records, error) {
	u := urlRecords(c.NetboxURL)
	u.RawQuery = rq.encode()
	rs, err := getMany[Record](c, u.String())
	if err != nil {
		return nil, err
	}
	if rq.Zone != nil {
		for k, r := range rs {
			if r.TTL == nil {
				rs[k].TTL = &rq.Zone.DefaultTTL
			}
		}
		return rs, nil
	}
	rrrs, err := resolveRecordTTLs(c, rs)
	if err != nil {
		return rs, err
	}
	return rrrs, nil
}

func (r *Record) ToRR() (dns.RR, error) {
	qt := dns.StringToType[r.Type]
	if qt == dns.TypeTXT {
		var txt []string
		if strings.HasPrefix(r.AbsoluteValue, `"`) {
			v := txtMultiValueRegexp.FindAllString(r.AbsoluteValue, -1)
			for i := range v {
				v[i] = strings.Trim(v[i], `"`)
				v[i] = strings.ReplaceAll(v[i], "\\r\\n", "")
				v[i] = strings.ReplaceAll(v[i], "\\n", "")
				v[i] = strings.TrimSpace(v[i])
				if v[i] != "" {
					txt = append(txt, v[i])
				}
			}
		} else {
			txt = append(txt, r.AbsoluteValue)
		}
		rr := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   r.FQDN,
				Ttl:    *r.TTL,
				Class:  dns.ClassINET,
				Rrtype: dns.TypeTXT,
			},
			Txt: txt,
		}
		return rr, nil
	}
	s := fmt.Sprintf(
		"%s %d IN %s %s",
		r.FQDN,
		*r.TTL,
		dns.TypeToString[qt],
		r.AbsoluteValue,
	)
	rr, err := dns.NewRR(s)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

func (r *Records) ToRRs() ([]dns.RR, error) {
	rrs := make([]dns.RR, 0, len(*r))
	for _, rr := range *r {
		rrr, err := rr.ToRR()
		if err != nil {
			return rrs, err
		}
		rrs = append(rrs, rrr)
	}
	return rrs, nil
}

func urlRecords(u *url.URL) *url.URL {
	return u.JoinPath("records", "/")
}

func GetRecordsQuery(c *Client, q *RecordQuery) ([]Record, error) {
	reqUrl := urlRecords(c.NetboxURL)
	reqUrl.RawQuery = q.encode()
	records, err := getMany[Record](c, reqUrl.String())
	if err != nil {
		return nil, err
	}
	if q.Zone != nil {
		for k, record := range records {
			if record.TTL == nil {
				records[k].TTL = &q.Zone.DefaultTTL
			}
		}
	} else {
		resolvedRecords, err := resolveRecordTTLs(c, records)
		if err != nil {
			return records, err
		}
		records = resolvedRecords
	}
	return records, nil
}

func resolveRecordTTLs(c *Client, r []Record) ([]Record, error) {
	zoneTTL := make(map[int]uint32)
	for k, record := range r {
		if record.TTL != nil {
			continue
		}
		if ttl, ok := zoneTTL[record.Zone.ID]; ok {
			r[k].TTL = &ttl
			continue
		}
		zoneUrl := urlZoneID(c.NetboxURL, record.Zone.ID)
		zone, err := get[Zone](c, zoneUrl.String())
		if err != nil {
			return r, err
		}
		zoneTTL[zone.ID] = zone.DefaultTTL
		r[k].TTL = &zone.DefaultTTL
	}
	return r, nil
}
