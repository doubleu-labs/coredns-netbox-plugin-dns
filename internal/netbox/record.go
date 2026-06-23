package netbox

import (
	"net/url"
	"strconv"
)

type Record struct {
	Type          string  `json:"type"`
	AbsoluteValue string  `json:"absolute_value"`
	TTL           *uint32 `json:"ttl"`
	Zone          Zone    `json:"zone"`
	FQDN          string  `json:"fqdn"`
}

type RecordQuery struct {
	FQDN string
	Name string
	Type []string
	Zone *Zone
}

func (rq *RecordQuery) Encode() string {
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

	if rq.Zone != nil {
		out.Set("zone_id", strconv.Itoa(rq.Zone.ID))
	}

	return out.Encode()
}

func urlRecords(u *url.URL) *url.URL {
	return u.JoinPath("records", "/")
}

func GetRecordsQuery(c *Client, q *RecordQuery) ([]Record, error) {
	reqUrl := urlRecords(c.NetboxURL)
	reqUrl.RawQuery = q.Encode()
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
