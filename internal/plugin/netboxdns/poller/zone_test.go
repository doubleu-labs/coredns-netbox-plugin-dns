package poller

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/cache"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
	"github.com/miekg/dns"
)

func Test_NewZonePollerRejectsNilClient(t *testing.T) {
	zoneCache := cache.NewCache(1)
	views := &core.Views{
		Include: []string{"."},
	}
	activeStatuses := []string{"active", "dynamic"}
	logger := log.NewWithPlugin("zone_poller_test")
	interval := time.Second

	if _, err := NewZonePoller(
		nil,
		zoneCache,
		views,
		activeStatuses,
		new(logger),
		interval,
	); err == nil {
		t.Fatal("expected error")
	}
}

func Test_NewZonePollerRejectsNilCache(t *testing.T) {
	client := new(api.Client)
	views := new(core.Views)
	var activeStatuses []string
	logger := log.NewWithPlugin("zone_poller_test")
	interval := time.Second

	if _, err := NewZonePoller(
		client,
		nil,
		views,
		activeStatuses,
		new(logger),
		interval,
	); err == nil {
		t.Fatal("expected error")
	}
}

func Test_NewZonePollerUsesDefaultInterval(t *testing.T) {
	client := new(api.Client)
	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	var activeStatuses []string
	logger := log.NewWithPlugin("zone_poller_test")
	interval := 0 * time.Second

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		activeStatuses,
		new(logger),
		interval,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}
	if zp.Poller.Interval != defaultZonePollerInterval {
		t.Fatalf("expected default interval; got %v", zp.Poller.Interval)
	}
}

func Test_NewZonePollerUsesExplicitInterval(t *testing.T) {
	client := new(api.Client)
	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")
	interval := 5 * time.Second

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		nil,
		new(logger),
		interval,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}
	if zp.Poller.Interval != interval {
		t.Fatalf("interval: want %v, got %v", interval, zp.Poller.Interval)
	}
}

func Test_NewZonePollerSetsPollFunc(t *testing.T) {
	client := new(api.Client)
	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")
	interval := time.Second

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		nil,
		new(logger),
		interval,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}
	if zp.Poller.PollFunc == nil {
		t.Fatal("expected poll func to be set")
	}
}

func Test_ZonePollerPollReturnsCanceledContextError(t *testing.T) {
	client := new(api.Client)
	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		nil,
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = zp.poll(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("poll error: want %v, got %v", context.Canceled, err)
	}
}

func Test_ZonePollerReturnsZoneAPIError(t *testing.T) {
	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "server error", http.StatusInternalServerError)
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}
	if pollErr := zp.poll(context.Background()); pollErr == nil {
		t.Fatal("expected poll error")
	}
}

func Test_ZonePollerStoresFetchedZoneRecordsInCache(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOAMName: struct {
								Name string `json:"name"`
							}{
								Name: "ns1.example.com",
							},
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				testutil.WritePoller(
					t,
					w,
					[]api.Record{
						{
							FQDN:          "example.com.",
							Type:          "NS",
							AbsoluteValue: "ns1.example.com.",
							TTL:           &ttl,
						},
						{
							FQDN:          "www.example.com.",
							Type:          "A",
							AbsoluteValue: "192.168.2.10",
							TTL:           &ttl,
						},
					},
				)
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	ch, err := zoneCache.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("transfer cache: %v", err)
	}

	var rrs []dns.RR
	for batch := range ch {
		rrs = slices.Concat(rrs, batch)
	}

	if !slices.ContainsFunc(
		rrs,
		func(rr dns.RR) bool {
			a, ok := rr.(*dns.A)
			return ok &&
				a.Hdr.Name == "www.example.com." &&
				a.A.String() == "192.168.2.10"
		},
	) {
		t.Fatalf("cached rrs do not contain expected a record: %v", rrs)
	}

	if !slices.ContainsFunc(
		rrs,
		func(rr dns.RR) bool {
			soa, ok := rr.(*dns.SOA)
			return ok &&
				soa.Hdr.Name == "example.com." &&
				soa.Serial == 123
		},
	) {
		t.Fatalf("cached rrs do not contain expected synthesized soa: %v", rrs)
	}
}

func Test_ZonePollerSynthesizesSOAMNameFromFirstNSRecord(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				testutil.WritePoller(
					t,
					w,
					[]api.Record{
						{
							FQDN:          "example.com.",
							Type:          "NS",
							AbsoluteValue: "ns1.example.com.",
							TTL:           &ttl,
						},
						{
							FQDN:          "example.com.",
							Type:          "NS",
							AbsoluteValue: "ns2.example.com.",
							TTL:           &ttl,
						},
					},
				)
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	ch, err := zoneCache.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("transfer cache: %v", err)
	}

	var gotSOA *dns.SOA
	for batch := range ch {
		for rr := range slices.Values(batch) {
			if soa, ok := rr.(*dns.SOA); ok {
				gotSOA = soa
			}
		}
	}

	if gotSOA == nil {
		t.Fatal("expected cached soa")
	}
	if gotSOA.Ns != "ns1.example.com." {
		t.Fatalf("soa mname: want %q; got %q", "ns1.example.com.", gotSOA.Ns)
	}
}

func Test_ZonePollerSynthesizeSOAWithoutMNameWhenNoNSRecordExist(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				testutil.WritePoller(
					t,
					w,
					[]api.Record{
						{
							FQDN:          "www.example.com.",
							Type:          "A",
							AbsoluteValue: "192.0.2.10",
							TTL:           &ttl,
						},
					},
				)
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	ch, err := zoneCache.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf("transfer cache: %v", err)
	}

	var gotSOA *dns.SOA
	for batch := range ch {
		for rr := range slices.Values(batch) {
			if soa, ok := rr.(*dns.SOA); ok {
				gotSOA = soa
			}
		}
	}

	if gotSOA == nil {
		t.Fatal("expected cached soa")
	}
	if gotSOA.Ns != "." {
		t.Fatalf("soa mname: want empty root %q; got %q", ".", gotSOA.Ns)
	}
}

func Test_ZonePollerContinuesWhenRecordFetchFails(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				http.Error(w, "server error", http.StatusInternalServerError)
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: want nil; got %v", pollErr)
	}

	if _, zoneCacheErr := zoneCache.Transfer(
		"example.com.",
		0,
	); zoneCacheErr != nil {
		t.Fatalf(
			"expected zone to be cached despite records error: %v",
			zoneCacheErr,
		)
	}
}

func Test_ZonePollerCachesMultipleZones(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
						{
							ID:         2,
							Name:       "example.org",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.org.",
							SOASerial:  456,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				switch r.URL.Query().Get("zone_id") {
				case "1":
					testutil.WritePoller(
						t,
						w,
						[]api.Record{
							{
								FQDN:          "www.example.com.",
								Type:          "A",
								AbsoluteValue: "192.0.2.10",
								TTL:           &ttl,
							},
						},
					)
				case "2":
					testutil.WritePoller(
						t,
						w,
						[]api.Record{
							{
								FQDN:          "www.example.org.",
								Type:          "A",
								AbsoluteValue: "192.0.2.20",
								TTL:           &ttl,
							},
						},
					)
				default:
					t.Fatalf(
						"unexpected zone_id query value: %q",
						r.URL.Query().Get("zone_id"),
					)
				}
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	gotNames := zoneCache.GetZoneNames()
	wantNames := []string{"example.com.", "example.org."}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("cached zone name: want %v, got %v", wantNames, gotNames)
	}

	if zoneCache.Size() != 2 {
		t.Fatalf("cached zone size: want %d, got %d", 2, zoneCache.Size())
	}
}

func Test_ZonePollerPassesStatusViewsAndRecordFilters(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				statuses := r.URL.Query()["status"]
				if !slices.Equal(statuses, []string{"active", "dynamic"}) {
					t.Fatalf(
						"status query: got %v; want %v",
						statuses,
						[]string{"active", "dynamic"},
					)
				}

				includeViews := r.URL.Query()["view"]
				if !slices.Equal(includeViews, []string{"internal"}) {
					t.Fatalf(
						"view query: got %v; want %v",
						includeViews,
						[]string{"internal"},
					)
				}

				excludeViews := r.URL.Query()["view__n"]
				if !slices.Equal(excludeViews, []string{"offsite"}) {
					t.Fatalf(
						"view__n query: got %v; want %v",
						excludeViews,
						[]string{"offsite"},
					)
				}

				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				if got := r.URL.Query().Get("zone_id"); got != "1" {
					t.Fatalf("zone_id query: got %q; want %q", got, "1")
				}
				if got := r.URL.Query()["type__n"]; !slices.Equal(
					got,
					[]string{"SOA"},
				) {
					t.Fatalf(
						"type__n query: got %v; want %v",
						got,
						[]string{"SOA"},
					)
				}

				testutil.WritePoller[api.Record](t, w, nil)
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views :=
		&core.Views{
			Include: []string{"internal"},
			Exclude: []string{"offsite"},
		}
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active", "dynamic"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}
}

func Test_ZonePollerContinuesWhenRecordsCannotConvertToRRs(t *testing.T) {
	ttl := uint32(300)

	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/zones/":
				testutil.WritePoller(
					t,
					w,
					[]api.Zone{
						{
							ID:         1,
							Name:       "example.com",
							DefaultTTL: ttl,
							SOATTL:     ttl,
							SOARName:   "hostmaster.example.com.",
							SOASerial:  123,
							SOARefresh: 3600,
							SOARetry:   600,
							SOAExpire:  86400,
							SOAMinimum: 300,
						},
					},
				)
			case "/records/":
				testutil.WritePoller(
					t,
					w,
					[]api.Record{
						{
							FQDN:          "www.example.com.",
							Type:          "A",
							AbsoluteValue: "192.0.2.10",
							TTL:           &ttl,
						},
						{
							FQDN:          "bad.example.com.",
							Type:          "A",
							AbsoluteValue: "not-an-ip",
							TTL:           &ttl,
						},
					},
				)
			default:
				http.NotFound(w, r)
			}
		},
	)
	defer closeServer()

	zoneCache := cache.NewCache(1)
	views := new(core.Views)
	logger := log.NewWithPlugin("zone_poller_test")

	zp, err := NewZonePoller(
		client,
		zoneCache,
		views,
		[]string{"active"},
		new(logger),
		time.Second,
	)
	if err != nil {
		t.Fatalf("new zone poller: %v", err)
	}

	if pollErr := zp.poll(context.Background()); pollErr != nil {
		t.Fatalf("poll error: want nil; got %v", pollErr)
	}

	ch, err := zoneCache.Transfer("example.com.", 0)
	if err != nil {
		t.Fatalf(
			"expected zone to be cached despite rr conversion error: %v",
			err,
		)
	}

	var rrs []dns.RR
	for batch := range ch {
		rrs = slices.Concat(rrs, batch)
	}

	if !slices.ContainsFunc(
		rrs,
		func(rr dns.RR) bool {
			soa, ok := rr.(*dns.SOA)
			return ok &&
				soa.Hdr.Name == "example.com." &&
				soa.Serial == 123
		},
	) {
		t.Fatalf("cached rrs do not contain expected synthesized soa: %v", rrs)
	}

	if slices.ContainsFunc(
		rrs,
		func(rr dns.RR) bool {
			return rr.Header().Name == "bad.example.com."
		},
	) {
		t.Fatalf("cached rrs unexpectedly contain malformed record: %v", rrs)
	}
}
