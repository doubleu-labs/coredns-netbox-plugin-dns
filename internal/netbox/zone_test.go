package netbox

import (
	"net/http"
	"net/url"
	"testing"
)

func TestUrlZones(t *testing.T) {
	base, _ := url.Parse("https://netbox.example.com/api/plugins/netbox-dns/")
	got := urlZones(base).String()
	want := "https://netbox.example.com/api/plugins/netbox-dns/zones/"
	if got != want {
		t.Errorf("urlZones = %q, want %q", got, want)
	}
}

func TestUrlZoneID(t *testing.T) {
	base, _ := url.Parse("https://netbox.example.com/api/plugins/netbox-dns/")
	got := urlZoneID(base, 42).String()
	want := "https://netbox.example.com/api/plugins/netbox-dns/zones/42/"
	if got != want {
		t.Errorf("urlZoneID = %q, want %q", got, want)
	}
}

func TestGetZones_Success(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, fixtureZonesSinglePage)
		},
	)

	zones, err := GetZones(client, "")
	if err != nil {
		t.Fatalf("GetZones: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("len = %d, want 2", len(zones))
	}
	if zones[0].DefaultTTL != 3600 || zones[1].DefaultTTL != 7200 {
		t.Errorf("default TTLs not parsed: %+v", zones)
	}
	if len(zones[0].Nameservers) != 2 || zones[0].Nameservers[0].Name != "ns1.example.com" {
		t.Errorf("nameservers not parsed: %+v", zones[0].Nameservers)
	}
}

func TestGetZones_Empty(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, fixtureZonesEmpty)
		},
	)
	zones, err := GetZones(client, "")
	if err != nil {
		t.Fatalf("GetZones: %v", err)
	}
	if len(zones) != 0 {
		t.Errorf("len = %d, want 0", len(zones))
	}
}

func TestGetZones_WithView_AppendsQueryParam(t *testing.T) {
	_, mux, client := mockNetbox(t)
	var gotView string
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, r *http.Request) {
			gotView = r.URL.Query().Get("view")
			writeJSON(w, fixtureZonesEmpty)
		},
	)

	if _, err := GetZones(client, "internal"); err != nil {
		t.Fatalf("GetZones: %v", err)
	}
	if gotView != "internal" {
		t.Errorf("?view query = %q, want %q", gotView, "internal")
	}
}

// GetZones always pins ?status=active so parked / deprecated / reserved
// zones are excluded from the normal serving paths.
func TestGetZones_AlwaysFiltersStatusActive(t *testing.T) {
	_, mux, client := mockNetbox(t)
	var gotStatus, gotView string
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, r *http.Request) {
			gotStatus = r.URL.Query().Get("status")
			gotView = r.URL.Query().Get("view")
			writeJSON(w, fixtureZonesEmpty)
		},
	)

	if _, err := GetZones(client, ""); err != nil {
		t.Fatalf("GetZones: %v", err)
	}
	if gotStatus != "active" {
		t.Errorf("?status = %q, want %q", gotStatus, "active")
	}
	if gotView != "" {
		t.Errorf(
			"?view should be empty when no view configured, got %q",
			gotView,
		)
	}
}

func TestGetZones_ParsesNestedView(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(
				w, `{
            "count": 1, "next": null, "previous": null,
            "results": [
                {
                    "id": 1, "name": "example.com", "default_ttl": 3600,
                    "nameservers": [{"name": "ns1.example.com"}],
                    "view": {"id": 2, "name": "internal"}
                }
            ]
        }`,
			)
		},
	)

	zones, err := GetZones(client, "internal")
	if err != nil {
		t.Fatalf("GetZones: %v", err)
	}
	if len(zones) != 1 || zones[0].View == nil {
		t.Fatalf("view not decoded: %+v", zones)
	}
	if zones[0].View.Name != "internal" || zones[0].View.ID != 2 {
		t.Errorf("view = %+v, want {ID:2 Name:internal}", zones[0].View)
	}
}

func TestGetZones_APIError(t *testing.T) {
	_, mux, client := mockNetbox(t)
	mux.HandleFunc(
		"/api/plugins/netbox-dns/zones/",
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		},
	)
	if _, err := GetZones(client, ""); err == nil {
		t.Fatal("expected error, got nil")
	}
}
