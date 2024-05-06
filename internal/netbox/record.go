package netbox

import (
	"net/url"
	"strconv"
)

type Record struct {
	Type  string  `json:"type"`
	Value string  `json:"value"`
	TTL   *uint32 `json:"ttl"`
	Zone  Zone    `json:"zone"`
	FQDN  string  `json:"fqdn"`
}

type RecordQuery struct {
	FQDN string
	Name string
	Type []string
	Zone *Zone
}

func (recordQuery *RecordQuery) Encode() string {
	out := url.Values{}

	if recordQuery.FQDN != "" {
		out.Set("fqdn", recordQuery.FQDN)
	}

	if recordQuery.Name != "" {
		out.Set("name", recordQuery.Name)
	}

	if len(recordQuery.Type) != 0 {
		for _, t := range recordQuery.Type {
			out.Add("type", t)
		}
	}

	if recordQuery.Zone != nil {
		out.Set("zone_id", strconv.Itoa(recordQuery.Zone.ID))
	}

	return out.Encode()
}

func urlRecords(netboxurl *url.URL) *url.URL {
	return netboxurl.JoinPath("records", "/")
}

func GetRecordsQuery(
	requestClient *APIRequestClient,
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
	requestClient *APIRequestClient,
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
