package testutil

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/miekg/dns"
)

func NewServeDNSTestClient(t *testing.T) *api.Client {
	t.Helper()
	u, err := url.Parse("http://127.0.0.1/")
	if err != nil {
		t.Fatalf("parse test url: %v", err)
	}
	return api.NewClient("", u)
}

type RecordingHandler struct {
	called bool
	Rcode  int
}

func (rh *RecordingHandler) Name() string {
	return "recording"
}

func (rh *RecordingHandler) ServeDNS(
	_ context.Context,
	_ dns.ResponseWriter,
	_ *dns.Msg,
) (int, error) {
	rh.called = true
	return rh.Rcode, nil
}

func (rh *RecordingHandler) Called() bool {
	return rh.called
}

type testAddr string

func (ta testAddr) Network() string {
	return string(ta)
}

func (ta testAddr) String() string {
	return string(ta)
}

type RecordingResponseWriter struct{}

func (*RecordingResponseWriter) LocalAddr() net.Addr {
	return testAddr("local")
}

func (*RecordingResponseWriter) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (*RecordingResponseWriter) WriteMsg(_ *dns.Msg) error {
	return nil
}

func (*RecordingResponseWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

func (*RecordingResponseWriter) Close() error {
	return nil
}

func (*RecordingResponseWriter) TsigStatus() error {
	return nil
}

func (*RecordingResponseWriter) TsigTimersOnly(_ bool) {
	return
}

func (*RecordingResponseWriter) Hijack() {
	return
}
