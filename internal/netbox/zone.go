package netbox

import (
	"net/url"
	"strconv"
)

type Zone struct {
	DefaultTTL  uint32     `json:"default_ttl"`
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	NameServers []SOAMName `json:"nameservers"`
}

type SOAMName struct {
	Name string `json:"name"`
}

func urlZones(netboxurl *url.URL) *url.URL {
	return netboxurl.JoinPath("zones", "/")
}

func urlZoneID(netboxurl *url.URL, id int) *url.URL {
	return netboxurl.JoinPath("zones", "/", strconv.Itoa(id), "/")
}

func GetZones(requestClient *APIRequestClient) ([]Zone, error) {
	requestUrl := urlZones(requestClient.NetboxURL)
	zones, err := getMany[Zone](requestClient, requestUrl.String())
	if err != nil {
		return nil, err
	}
	return zones, nil
}
