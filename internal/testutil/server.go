package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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
	addr     string
	instance *caddy.Instance
}

// Send sends a DNS message to the server and returns the response.
func (ts *TestServer) Send(m *dns.Msg) (r *dns.Msg, err error) {
	dnsClient := new(dns.Client)
	addr := net.JoinHostPort("127.0.0.1", ts.addr)
	r, _, err = dnsClient.Exchange(m, addr)
	return
}

func (ts *TestServer) Transfer(m *dns.Msg) (chan *dns.Envelope, error) {
	client := new(dns.Transfer)
	addr := net.JoinHostPort("127.0.0.1", ts.addr)
	return client.In(m, addr)
}

func (ts *TestServer) Close(t *testing.T) {
	err := ts.instance.Stop()
	if err != nil {
		t.Errorf("Failed to stop coredns instance: %v", err)
	}
	ts.instance.ShutdownCallbacks()
	ts.instance.Wait()
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

	testServer := new(TestServer)

	serverInstance, addr := startOnSharedDynamicPort(t, serverBlockContent)

	testServer.addr = addr
	testServer.instance = serverInstance

	return testServer
}

func startOnSharedDynamicPort(t *testing.T, serverBlockContents string) (
	*caddy.Instance,
	string,
) {
	t.Helper()
	const maxAttempts = 20
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		addr := reserveSharedAddr(t)
		corefile := fmt.Sprintf(".:%s {\n\t%s\n}", addr, serverBlockContents)
		serverInstance, err := caddy.Start(
			caddy.CaddyfileInput{
				ServerTypeName: "dns",
				Contents:       []byte(corefile),
			},
		)
		if err != nil {
			lastErr = err
			if isAddrInUseErr(err) {
				continue
			}
			t.Fatalf("failed to start server instance: %v", err)
		}
		servers := serverInstance.Servers()
		if len(servers) == 0 {
			_ = serverInstance.Stop()
			serverInstance.ShutdownCallbacks()
			t.Fatal("no servers started")
		}
		return serverInstance, addr
	}
	t.Fatalf(
		"failed to start server instance on a shared tcp/udp test port after "+
			"%d attempts: %v",
		maxAttempts,
		lastErr,
	)
	return nil, ""
}

func reserveSharedAddr(t *testing.T) string {
	t.Helper()
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test tcp port: %v", err)
	}
	defer func() {
		if closeErr := tcpListener.Close(); closeErr != nil {
			t.Fatalf("failed to close test tcp listener: %v", closeErr)
		}
	}()
	tcpAddr, ok := tcpListener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("failed to get TCP address from listener")
	}
	udpAddr := &net.UDPAddr{
		IP:   tcpAddr.IP,
		Port: tcpAddr.Port,
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf(
			"failed to reserve matching test udp port %d: %v",
			udpAddr.Port,
			err,
		)
	}
	defer func() {
		if closeErr := udpConn.Close(); closeErr != nil {
			t.Fatalf("failed to close test udp listener: %v", closeErr)
		}
	}()
	return fmt.Sprintf("%d", tcpAddr.Port)
}

func isAddrInUseErr(err error) bool {
	for err != nil {
		if opErr := new(net.OpError); errors.As(err, &opErr) {
			if strings.Contains(opErr.Err.Error(), "address already in use") {
				return true
			}
		}
		if strings.Contains(err.Error(), "address already in use") {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
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
	var apiToken token
	decoderErr := json.NewDecoder(response.Body).Decode(&apiToken)
	if decoderErr != nil {
		t.Fatal("provision api token:", decoderErr.Error())
	}
	hostURL.Path = ""
	return hostURL.String(), apiToken.String()
}
