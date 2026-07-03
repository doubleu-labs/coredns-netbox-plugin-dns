package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/coredns/caddy"
	"github.com/miekg/dns"
)

var (
	serverBlockContentURLRegex   *regexp.Regexp
	serverBlockContentTokenRegex *regexp.Regexp

	netboxTestInstanceUriHost = "127.0.0.1:9999"

	netboxProbeOnce sync.Once
	netboxProbeErr  error
)

func init() {
	serverBlockContentURLRegex = regexp.MustCompile(`<!URL>`)
	serverBlockContentTokenRegex = regexp.MustCompile(`<!TOKEN>`)
}

type token struct {
	Key   string `json:"key"`
	Token string `json:"token"`
}

func (t *token) String() string {
	return fmt.Sprintf("nbt_%s.%s", t.Key, t.Token)
}

// TestServer is a local ephemeral CoreDNS server for testing.
type TestServer struct {
	addr       string
	instance   *caddy.Instance
	clientConn net.Conn
	serverConn net.Conn
}

// Send sends a DNS message to the server and returns the response.
func (ts *TestServer) Send(m *dns.Msg) (r *dns.Msg, err error) {
	dnsClient := new(dns.Client)
	r, _, err = dnsClient.Exchange(m, ts.addr)
	return
}

func (ts *TestServer) Close(t *testing.T) {
	ts.closeServer(t)
	go ts.closeConn(t, ts.clientConn)
	go ts.closeConn(t, ts.serverConn)
}

func (ts *TestServer) closeConn(t *testing.T, conn net.Conn) {
	if conn == nil {
		return
	}
	err := conn.Close()
	if err != nil {
		t.Errorf("Failed to close connection: %v", err)
	}
}

func (ts *TestServer) closeServer(t *testing.T) {
	err := ts.instance.Stop()
	if err != nil {
		t.Errorf("Failed to stop coredns instance: %v", err)
	}
	ts.instance.ShutdownCallbacks()
}

func probeNetbox(t *testing.T) {
	t.Helper()
	netboxProbeOnce.Do(
		func() {
			conn, err := net.DialTimeout(
				"tcp",
				netboxTestInstanceUriHost,
				300*time.Millisecond,
			)
			if err != nil {
				netboxProbeErr = err
				return
			}
			_ = conn.Close()
		},
	)
	if netboxProbeErr != nil {
		t.Skipf(
			"netbox integration test: %s unreachable (%v); start it "+
				"with `just instance-start`",
			netboxTestInstanceUriHost, netboxProbeErr,
		)
	}
}

// NewTestServer creates a new TestServer using the provided contents of a
// server block.
func NewTestServer(t *testing.T, serverBlockContent string) *TestServer {
	t.Helper()

	probeNetbox(t)

	hostURL, hostToken := GetTokenAndUrl(t)

	serverBlockContent = serverBlockContentURLRegex.ReplaceAllString(
		serverBlockContent,
		hostURL,
	)
	serverBlockContent = serverBlockContentTokenRegex.ReplaceAllString(
		serverBlockContent,
		hostToken,
	)

	corefile := fmt.Sprintf(
		`.:0 {
	%s
	}`, serverBlockContent,
	)

	testServer := new(TestServer)
	testServer.serverConn, testServer.clientConn = net.Pipe()

	serverInstance, err := caddy.Start(
		caddy.CaddyfileInput{
			ServerTypeName: "dns",
			Contents:       []byte(corefile),
		},
	)
	if err != nil {
		t.Fatalf("Failed to start coredns instance: %v", err)
	}

	servers := serverInstance.Servers()
	if len(servers) == 0 {
		t.Fatalf("No servers started")
	}

	return &TestServer{
		addr:     servers[0].LocalAddr().String(),
		instance: serverInstance,
	}
}

func GetTokenAndUrl(t *testing.T) (string, string) {
	probeNetbox(t)

	if netboxProbeErr != nil {
		return "", ""
	}
	body := bytes.NewBuffer([]byte(`{"username":"admin","password":"admin"}`))
	hostURL := &url.URL{
		Scheme: "http",
		Host:   netboxTestInstanceUriHost,
	}
	request := &http.Request{
		Method:        "POST",
		URL:           hostURL,
		Body:          io.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
	}
	request.URL.Path = "/api/users/tokens/provision/"
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal("provision api token:", err.Error())
	}
	var token token
	decoderErr := json.NewDecoder(response.Body).Decode(&token)
	if decoderErr != nil {
		t.Fatal("provision api token:", decoderErr.Error())
	}
	hostURL.Path = ""
	return hostURL.String(), token.String()
}
