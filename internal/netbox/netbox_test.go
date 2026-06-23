package netbox

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestDoGet_SuccessSetsAuthAndUserAgent(t *testing.T) {
	_, mux, client := mockNetbox(t)

	var gotAuth, gotUA string
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotUA = r.Header.Get("User-Agent")
			writeJSON(w, fixtureZonesEmpty)
		},
	)

	resp, err := doGet(client, client.NetboxURL.JoinPath("zones", "/").String())
	if err != nil {
		t.Fatalf("doGet: %v", err)
	}
	defer resp.Body.Close()

	if want := "Bearer test-token"; gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
	if gotUA != "netboxdns-unit-tests" {
		t.Errorf(
			"User-Agent header = %q, want %q",
			gotUA,
			"netboxdns-unit-tests",
		)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestDoGet_ConnectionError(t *testing.T) {
	// Build a client pointing to a closed listener.
	bogus, _ := url.Parse("http://127.0.0.1:1") // port 1 should refuse
	client := &Client{
		Client:    http.DefaultClient,
		NetboxURL: bogus,
		Token:     "x",
	}
	_, err := doGet(client, "http://127.0.0.1:1/nope")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestResponseError(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"200 ok", http.StatusOK, false},
		{"401 unauthorized", http.StatusUnauthorized, true},
		{"403 forbidden", http.StatusForbidden, true},
		{"404 not found", http.StatusNotFound, true},
		{"500 server error", http.StatusInternalServerError, true},
	}
	for _, tc := range cases {
		t.Run(
			tc.name, func(t *testing.T) {
				resp := &http.Response{
					StatusCode: tc.status,
					Status:     fmt.Sprintf("%d", tc.status),
				}
				err := responseError(resp)
				if tc.wantErr && err == nil {
					t.Errorf("status %d: want error, got nil", tc.status)
				}
				if !tc.wantErr && err != nil {
					t.Errorf("status %d: want nil, got %v", tc.status, err)
				}
			},
		)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/1/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, `{"id": "not-an-int"}`)
		},
	)
	_, err := get[Zone](
		client,
		client.NetboxURL.JoinPath("zones", "/", "1", "/").String(),
	)
	if err == nil || !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("want unmarshal error, got %v", err)
	}
}

func TestGet_5xxPropagatesError(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/1/",
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		},
	)
	_, err := get[Zone](
		client,
		client.NetboxURL.JoinPath("zones", "/", "1", "/").String(),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMany_SinglePage(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, fixtureZonesSinglePage)
		},
	)

	zones, err := getMany[Zone](
		client,
		client.NetboxURL.JoinPath("zones", "/").String(),
	)
	if err != nil {
		t.Fatalf("getMany: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("len = %d, want 2", len(zones))
	}
	if zones[0].Name != "example.com" || zones[1].Name != "example.org" {
		t.Errorf("unexpected zones: %+v", zones)
	}
}

func TestGetMany_MultiPage(t *testing.T) {
	srv, mux, client := mockNetbox(t)

	page1 := fmt.Sprintf(
		`{
        "count": 3,
        "next": "%s/api/plugins/netbox-dns/zones/?page=2",
        "previous": null,
        "results": [
            {"id": 1, "name": "a.example.com", "default_ttl": 60, "nameservers": []}
        ]
    }`, srv.URL,
	)

	page2 := fmt.Sprintf(
		`{
        "count": 3,
        "next": "%s/api/plugins/netbox-dns/zones/?page=3",
        "previous": null,
        "results": [
            {"id": 2, "name": "b.example.com", "default_ttl": 60, "nameservers": []}
        ]
    }`, srv.URL,
	)

	page3 := `{
        "count": 3,
        "next": null,
        "previous": null,
        "results": [
            {"id": 3, "name": "c.example.com", "default_ttl": 60, "nameservers": []}
        ]
    }`

	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("page") {
			case "":
				writeJSON(w, page1)
			case "2":
				writeJSON(w, page2)
			case "3":
				writeJSON(w, page3)
			default:
				http.NotFound(w, r)
			}
		},
	)

	zones, err := getMany[Zone](
		client,
		client.NetboxURL.JoinPath("zones", "/").String(),
	)
	if err != nil {
		t.Fatalf("getMany: %v", err)
	}
	if len(zones) != 3 {
		t.Fatalf("len = %d, want 3", len(zones))
	}
	want := []string{"a.example.com", "b.example.com", "c.example.com"}
	for i, z := range zones {
		if z.Name != want[i] {
			t.Errorf("zone[%d] = %q, want %q", i, z.Name, want[i])
		}
	}
}

func TestGetMany_EmptyResults(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, fixtureZonesEmpty)
		},
	)
	zones, err := getMany[Zone](
		client,
		client.NetboxURL.JoinPath("zones", "/").String(),
	)
	if err != nil {
		t.Fatalf("getMany: %v", err)
	}
	if len(zones) != 0 {
		t.Fatalf("len = %d, want 0", len(zones))
	}
}

func TestGetMany_PropagatesHTTPError(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusUnauthorized)
		},
	)
	_, err := getMany[Zone](
		client,
		client.NetboxURL.JoinPath("zones", "/").String(),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
