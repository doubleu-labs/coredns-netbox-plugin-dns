package testutil

import (
	"bytes"
	"io"
	"log"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/coredns/caddy"
)

type SetupTest struct {
	Name            string
	ServerBlock     string
	ServerBlockKeys []string
	Want            string
	WantErr         bool
}

type SetupFunc func(c *caddy.Controller) error

func newTestController(sb string, sbk []string) *caddy.Controller {
	c := caddy.NewTestController("dns", sb)
	if sbk == nil {
		c.ServerBlockKeys = []string{"."}
	} else {
		c.ServerBlockKeys = sbk
	}
	return c
}

func RunSetupTests(t *testing.T, tt []SetupTest, s SetupFunc, st *bool) {
	*st = true
	for test := range slices.Values(tt) {
		t.Run(
			test.Name,
			func(t *testing.T) {
				c := newTestController(test.ServerBlock, test.ServerBlockKeys)
				if err := s(c); (err != nil) != test.WantErr {
					t.Errorf("setup error: %v; wantErr: %t", err, test.WantErr)
				}
			},
		)
	}
}

func RunSetupLogTests(
	t *testing.T,
	tests []SetupTest,
	setup SetupFunc,
	setupTest *bool,
) {
	*setupTest = true

	for test := range slices.Values(tests) {
		t.Run(
			test.Name,
			func(t *testing.T) {
				runSetupLogTest(t, test, setup)
			},
		)
	}
}

func runSetupLogTest(t *testing.T, test SetupTest, setup SetupFunc) {
	got := captureSetupLog(
		t, func() {
			c := newTestController(test.ServerBlock, test.ServerBlockKeys)
			if err := setup(c); err != nil {
				t.Errorf("setup error: %v", err)
			}
		},
	)

	want := strings.TrimSpace(test.Want)
	if strings.TrimSpace(got) != want {
		t.Errorf("got: %q; want: %q", got, test.Want)
	}
}

func captureSetupLog(t *testing.T, fn func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer func(r *os.File) {
		if closerErr := r.Close(); closerErr != nil {
			t.Fatalf("failed to close pipe: %v", closerErr)
		}
	}(reader)

	oldFlags := log.Flags()
	log.SetOutput(writer)
	log.SetFlags(0)
	defer func(flags int) {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
	}(oldFlags)

	fn()

	if closerErr := writer.Close(); closerErr != nil {
		t.Fatalf("failed to close pipe: %v", closerErr)
	}

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, reader); copyErr != nil {
		t.Fatalf("failed to copy pipe: %v", copyErr)
	}

	return buf.String()
}
