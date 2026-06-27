package api

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPClientTimeout = 5 * time.Second
	defaultUserAgent         = "coredns-netbox-plugin-dns"

	metricsEndpointLabelPathPrefix = "/api/plugins/netbox-dns/"
	metricsFallbackEndpointLabel   = "other"
)

var NetboxClient *Client

type Client struct {
	client    *http.Client
	token     *token
	netboxURL *url.URL
	userAgent string
}

// NewClient returns a new Netbox API client
func NewClient(token string, u *url.URL) *Client {
	NetboxClient = &Client{
		client: &http.Client{
			Timeout: defaultHTTPClientTimeout,
			Transport: &instrumentedTransport{
				requestsDuration: newRequestDurationMetric(),
				requestsTotal:    newRequestsTotalMetric(),
			},
		},
		netboxURL: u,
		token:     newToken(token),
		userAgent: defaultUserAgent,
	}
	return NetboxClient
}

// Do executes an HTTP request against the Netbox API.
// `Authorization` and `User-Agent` headers are set automatically. Existing
// values are not modified.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set(
		"Authorization",
		fmt.Sprintf("Token %s", c.token.raw),
	)
	req.Header.Set("User-Agent", c.userAgent)
	return c.client.Do(req)
}

// SetTransport sets the HTTP transport to be used by the client.
// If existing transport is set and its TLSClientConfig is set, then it is
// preserved and passed to the new transport.
func (c *Client) SetTransport(t http.RoundTripper) {
	if c.client.Transport == nil {
		c.client.Transport = t
		return
	}

	var tc *tls.Config
	if c.client.Transport.(*http.Transport).TLSClientConfig != nil {
		tc = c.client.Transport.(*http.Transport).TLSClientConfig
	}

	c.client.Transport = t

	if tc != nil {
		c.client.Transport.(*http.Transport).TLSClientConfig = tc
	}
}

// SetTLSConfig sets the TLS configuration to be used by the client.
func (c *Client) SetTLSConfig(t *tls.Config) {
	c.client.Transport.(*http.Transport).TLSClientConfig = t
}

// SetUserAgent sets the User-Agent header to be used by the client.
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
}

// SetTimeout sets the timeout used to be by the client.
func (c *Client) SetTimeout(d time.Duration) {
	c.client.Timeout = d
}

type instrumentedTransport struct {
	http.RoundTripper
	requestsDuration *requestDurationMetric
	requestsTotal    *requestsTotalMetric
}

// RoundTrip implements the http.RoundTripper interface.
func (it *instrumentedTransport) RoundTrip(req *http.Request) (
	*http.Response,
	error,
) {
	l := it.endpointLabel(req.URL.Path)
	s := time.Now()
	resp, err := it.RoundTrip(req)
	it.requestsDuration.Observe(l, time.Since(s).Seconds())
	if err != nil {
		it.requestsTotal.IncError(l)
		return resp, err
	}
	it.requestsTotal.IncStatusCode(l, resp.StatusCode)
	return resp, nil
}

func (*instrumentedTransport) endpointLabel(p string) string {
	if !strings.HasPrefix(p, metricsEndpointLabelPathPrefix) {
		return metricsFallbackEndpointLabel
	}
	r := strings.TrimPrefix(p, metricsEndpointLabelPathPrefix)
	if i := strings.IndexByte(r, '/'); i >= 0 {
		r = r[:i]
	}
	if r == "" {
		return metricsFallbackEndpointLabel
	}
	return r
}
