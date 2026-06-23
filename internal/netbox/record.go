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

func urlRecords(netboxurl *url.URL) *url.URL {
	return netboxurl.JoinPath("records", "/")
}

func GetRecordsQuery(
	requestClient *Client,
	query *RecordQuery,
) ([]Record, error) {
	requestUrl := urlRecords(requestClient.NetboxURL)
	requestUrl.RawQuery = query.Encode()
	records, err := getMany[Record](requestClient, requestUrl.String())
	if err != nil {
		return nil, err
	}
	if query.Zone != nil {
		for k, record := range records {
			if record.TTL == nil {
				records[k].TTL = &query.Zone.DefaultTTL
			}
		}
	} else {
		resolvedRecords, err := resolveRecordTTLs(requestClient, records)
		if err != nil {
			return records, err
		}
		records = resolvedRecords
	}
	return records, nil
}

func resolveRecordTTLs(
	requestClient *Client,
	records []Record,
) ([]Record, error) {
	zoneTTL := make(map[int]uint32)
	for k, record := range records {
		if record.TTL != nil {
			continue
		}
		if ttl, ok := zoneTTL[record.Zone.ID]; ok {
			records[k].TTL = &ttl
			continue
		}
		zoneUrl := urlZoneID(requestClient.NetboxURL, record.Zone.ID)
		zone, err := get[Zone](requestClient, zoneUrl.String())
		if err != nil {
			return records, err
		}
		zoneTTL[zone.ID] = zone.DefaultTTL
		records[k].TTL = &zone.DefaultTTL
	}
	return records, nil
}
