package poller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

func newViewPollerMockClient(t *testing.T, v []api.View) (*api.Client, func()) {
	t.Helper()
	return viewPollerMockHandler(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			writeViewPollerViews(t, w, v)
		},
	)
}

func viewPollerMockHandler(t *testing.T, handler http.HandlerFunc) (
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

func writeViewPollerViews(
	t *testing.T,
	w http.ResponseWriter,
	views []api.View,
) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(
		struct {
			Count    int        `json:"count"`
			Next     string     `json:"next"`
			Previous string     `json:"previous"`
			Results  []api.View `json:"results"`
		}{
			Count:   len(views),
			Results: views,
		},
	); err != nil {
		t.Fatalf("encode views response: %v", err)
	}
}

func Test_NewViewPollerRejectsNilClient(t *testing.T) {
	views := &core.Views{
		Include: []string{"internal"},
	}

	if _, err := NewViewPoller(nil, views, time.Second); err == nil {
		t.Fatal("expected error")
	}
}

func Test_NewViewPollerRejectsNilViews(t *testing.T) {
	client := &api.Client{}

	if _, err := NewViewPoller(client, nil, time.Second); err == nil {
		t.Fatal("expected error")
	}
}

func Test_NewViewPollerUsesDefaultInterval(t *testing.T) {
	client := &api.Client{}

	views := &core.Views{
		Include: []string{"internal"},
	}
	vp, err := NewViewPoller(client, views, 0)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}
	if vp.interval != defaultViewPollerInterval {
		t.Fatalf("expected default interval; got %v", vp.interval)
	}
}

func Test_ViewPollerRemovesMissingIncludeViews(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "internal",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal", "offsite"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	if pollErr := vp.poll(context.Background(), time.Now()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	if !vp.CanResolve() {
		t.Fatalf("expected view poller to be able to resolve")
	}

	gotViews := vp.Views()
	if !slices.Equal(gotViews.Include, []string{"internal"}) {
		t.Fatalf(
			"runtime include views: %v; want %v",
			gotViews.Include,
			[]string{"internal"},
		)
	}
	if !slices.Equal(vp.missingInclude(), []string{"offsite"}) {
		t.Fatalf(
			"missing include views: %v; want %v",
			vp.missingInclude(),
			[]string{"offsite"},
		)
	}
	if vp.lastErr() != nil {
		t.Fatalf("last error: %v; want nil", vp.lastErr())
	}
}

func Test_ViewPollerDisablesWhenNoIncludeViewsRemain(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "public",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal", "offsite"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	if pollErr := vp.poll(context.Background(), time.Now()); pollErr == nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	if vp.CanResolve() {
		t.Fatalf("expected view poller to disable resolution")
	}

	gotViews := vp.Views()
	if len(gotViews.Include) != 0 {
		t.Fatalf("runtime include views: %v; want empty", gotViews.Include)
	}

	wantMissing := []string{"internal", "offsite"}
	if !slices.Equal(vp.missingInclude(), wantMissing) {
		t.Fatalf(
			"missing include views: %v; want %v",
			vp.missingInclude(),
			wantMissing,
		)
	}

	if vp.lastErr() == nil {
		t.Fatal("expected last error to be non-nil")
	}
}

func Test_ViewPollerRemoveMissingExcludeViewsWithoutDisable(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "internal",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Exclude: []string{"internal", "offsite"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	if pollErr := vp.poll(context.Background(), time.Now()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	if !vp.CanResolve() {
		t.Fatalf("expected view poller to be able to resolve")
	}

	gotViews := vp.Views()
	if !slices.Equal(gotViews.Exclude, []string{"internal"}) {
		t.Fatalf(
			"runtime exclude views: %v; want %v",
			gotViews.Exclude,
			[]string{"internal"},
		)
	}
	if !slices.Equal(vp.missingExclude(), []string{"offsite"}) {
		t.Fatalf(
			"missing exclude views: %v; want %v",
			vp.missingExclude(),
			[]string{"offsite"},
		)
	}
	if vp.lastErr() != nil {
		t.Fatalf("last error: %v; want nil", vp.lastErr())
	}
}

func Test_ViewPollerKeepPreviousStateOnAPIError(t *testing.T) {
	status := http.StatusOK
	client, closeServer := viewPollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			if status == http.StatusOK {
				writeViewPollerViews(
					t, w, []api.View{
						{
							Name: "internal",
						},
					},
				)
			}
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	if pollErr := vp.poll(context.Background(), time.Now()); pollErr != nil {
		t.Fatalf("initial poll error: %v", pollErr)
	}

	status = http.StatusInternalServerError

	if pollErr := vp.poll(context.Background(), time.Now()); pollErr == nil {
		t.Fatal("second poll error is nil; want err")
	}

	if !vp.CanResolve() {
		t.Fatal("expected view poller to be able to resolve")
	}

	gotViews := vp.Views()
	if !slices.Equal(gotViews.Include, []string{"internal"}) {
		t.Fatalf(
			"runtime include views: %v; want %v",
			gotViews.Include,
			[]string{"internal"},
		)
	}
	if vp.lastErr() == nil {
		t.Fatal("expected last error to be non-nil")
	}
}

func Test_ViewPollerReturnsClone(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "internal",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	if pollErr := vp.poll(context.Background(), time.Now()); pollErr != nil {
		t.Fatalf("poll error: %v", pollErr)
	}

	gotViews := vp.Views()
	gotViews.Include[0] = "mutated"

	gotViews = vp.Views()
	if !slices.Equal(gotViews.Include, []string{"internal"}) {
		t.Fatalf(
			"runtime include views: %v; want %v",
			gotViews.Include,
			[]string{"internal"},
		)
	}
}

func Test_ViewPollerStartAlreadyStarted(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "internal",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	vp.Start()
	vp.Start()
}

func Test_ViewPollerStopAlreadyStopped(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "internal",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	vp.Stop()
	vp.Stop()
}

func Test_ViewPollerStartStop(t *testing.T) {
	client, closeServer := newViewPollerMockClient(
		t,
		[]api.View{
			{
				Name: "internal",
			},
		},
	)
	defer closeServer()

	views := &core.Views{
		Include: []string{"internal"},
	}
	vp, err := NewViewPoller(client, views, time.Second)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	vp.Start()
	time.Sleep(3 * time.Second)
	vp.Stop()
}
