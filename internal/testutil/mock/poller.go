package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
)

func NewPollerClient[T any](t *testing.T, results []T) (
	*api.Client,
	func(),
) {
	t.Helper()
	return PollerHandler(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			WritePoller(t, w, results)
		},
	)
}

func PollerHandler(t *testing.T, handler http.HandlerFunc) (
	*api.Client,
	func(),
) {
	t.Helper()
	server := httptest.NewServer(handler)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		t.Fatalf("parse server URL: %v", err)
	}
	client := api.NewClient("", serverURL)
	return client, server.Close
}

func WritePoller[T any](t *testing.T, w http.ResponseWriter, results []T) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(
		struct {
			Count    int    `json:"count"`
			Next     string `json:"next"`
			Previous string `json:"previous"`
			Results  []T    `json:"results"`
		}{
			Count:   len(results),
			Results: results,
		},
	); err != nil {
		t.Fatalf("encode poller response: %v", err)
	}
}
