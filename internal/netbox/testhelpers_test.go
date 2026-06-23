package netbox

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// mockNetbox spins up an httptest.Server and returns it together with a
// preconfigured Client whose NetboxURL points at
// <server>/api/plugins/netbox-dns/ (the same path Parse() builds in production).
//
// Tests register their handlers on the returned *http.ServeMux. The caller
// must Close() the server (use t.Cleanup).
func mockNetbox(t *testing.T) (*httptest.Server, *http.ServeMux, *Client) {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL + "/api/plugins/netbox-dns/")
	if err != nil {
		t.Fatalf("parse mock url: %v", err)
	}
	client := &Client{
		Client:    srv.Client(),
		NetboxURL: base,
		UserAgent: "netboxdns-unit-tests",
	}
	client.SetToken("test-token")
	return srv, mux, client
}

// writeJSON is a small helper that writes a 200 response with a JSON body.
func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// Universal fixtures used across the netbox package tests. These are
// deliberately generic (example.com / example.org / RFC1918 + RFC3849) so
// they never accidentally reference real infrastructure.
const (
	fixtureZonesSinglePage = `{
        "count": 2,
        "next": null,
        "previous": null,
        "results": [
            {
                "id": 1,
                "name": "example.com",
                "default_ttl": 3600,
                "nameservers": [
                    {"name": "ns1.example.com"},
                    {"name": "ns2.example.com"}
                ]
            },
            {
                "id": 2,
                "name": "example.org",
                "default_ttl": 7200,
                "nameservers": [
                    {"name": "ns1.example.org"}
                ]
            }
        ]
    }`

	fixtureZonesEmpty = `{"count": 0, "next": null, "previous": null, "results": []}`

	// Single zone returned by /zones/<id>/ — used by resolveRecordTTLs.
	fixtureZoneByID1 = `{
        "id": 1,
        "name": "example.com",
        "default_ttl": 3600,
        "nameservers": [{"name": "ns1.example.com"}]
    }`
)
