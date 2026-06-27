package netbox

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/metrics"
)

type token struct {
	raw string
}

func newToken(s string) *token {
	return &token{raw: s}
}

const (
	redacted                 = "[REDACTED]"
	defaultHTTPClientTimeout = time.Second * 5
)

// Format redacts the token when printed.
func (t *token) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(redacted))
}

// Client for interacting with the Netbox API.
type Client struct {
	*http.Client

	token     *token
	NetboxURL *url.URL
	UserAgent string
}

func NewClient(m *metrics.Metrics) *Client {
	return &Client{
		Client: &http.Client{
			Timeout: defaultHTTPClientTimeout,
			Transport: metrics.NewInstrumentedTransport(
				http.DefaultTransport,
				m,
			),
		},
	}
}

func (c *Client) SetTLSConfig(tlsConfig *tls.Config) {
	c.Transport.(*http.Transport).TLSClientConfig = tlsConfig
}

func (c *Client) SetTimeout(d time.Duration) {
	c.Client.Timeout = d
}

// SetToken sets the token for the Client.
func (c *Client) SetToken(t string) {
	c.token = newToken(t)
}

// HasToken returns true if the Client has a token set.
func (c *Client) HasToken() bool {
	if c.token == nil {
		return false
	}
	return c.token.raw != ""
}
