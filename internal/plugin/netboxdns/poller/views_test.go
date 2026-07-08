package poller

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
)

func Test_NewViewPollerRejectsNilClient(t *testing.T) {
	views := new(core.Views)
	interval := time.Second

	if _, err := NewViewPoller(nil, views, interval); err == nil {
		t.Fatal("expected error")
	}
}

func Test_NewViewPollerRejectsNilViews(t *testing.T) {
	client := new(api.Client)
	interval := time.Second

	if _, err := NewViewPoller(client, nil, interval); err == nil {
		t.Fatal("expected error")
	}
}

func Test_NewViewPollerUsesDefaultInterval(t *testing.T) {
	client := new(api.Client)
	views := new(core.Views)
	interval := 0 * time.Second

	vp, err := NewViewPoller(client, views, interval)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}
	if vp.Poller.Interval != defaultViewPollerInterval {
		t.Fatalf("expected default interval; got %v", vp.Poller.Interval)
	}
}

func Test_NewViewPollerUsesExplicitInterval(t *testing.T) {
	client := new(api.Client)
	views := new(core.Views)
	interval := 5 * time.Second

	vp, err := NewViewPoller(client, views, interval)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}
	if vp.Poller.Interval != interval {
		t.Fatalf("interval; want %v; got %v", interval, vp.Poller.Interval)
	}
}

func Test_NewViewPollerSetsPollFunc(t *testing.T) {
	client := new(api.Client)
	views := new(core.Views)
	interval := time.Second

	vp, err := NewViewPoller(client, views, interval)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}
	if vp.Poller.PollFunc == nil {
		t.Fatalf("expected poll func to be set")
	}
}

func Test_ViewPollerPollReturnsCanceledContextError(t *testing.T) {
	client := new(api.Client)
	views := new(core.Views)
	interval := time.Second

	vp, err := NewViewPoller(client, views, interval)
	if err != nil {
		t.Fatalf("new view poller: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = vp.poll(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("poll error: want %v, got %v", context.Canceled, err)
	}
}

func Test_ViewPollerRemovesMissingIncludeViews(t *testing.T) {
	client, closeServer := testutil.NewPollerMockClient(
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

	if pollErr := vp.poll(context.Background()); pollErr != nil {
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
	client, closeServer := testutil.NewPollerMockClient(
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

	if pollErr := vp.poll(context.Background()); pollErr == nil {
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
	client, closeServer := testutil.NewPollerMockClient(
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

	if pollErr := vp.poll(context.Background()); pollErr != nil {
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
	client, closeServer := testutil.PollerMockHandler(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			if status == http.StatusOK {
				testutil.WritePoller(
					t,
					w,
					[]api.View{
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

	if pollErr := vp.poll(context.Background()); pollErr != nil {
		t.Fatalf("initial poll error: %v", pollErr)
	}

	status = http.StatusInternalServerError

	if pollErr := vp.poll(context.Background()); pollErr == nil {
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
	client, closeServer := testutil.NewPollerMockClient(
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

	if pollErr := vp.poll(context.Background()); pollErr != nil {
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
	client, closeServer := testutil.NewPollerMockClient(
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
	client, closeServer := testutil.NewPollerMockClient(
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
	client, closeServer := testutil.NewPollerMockClient(
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
