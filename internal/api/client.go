package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPClientTimeout = 2 * time.Second
	defaultUserAgent         = "coredns-netbox-plugin-dns"

	metricsFallbackEndpointLabel = "other"
)

type Client struct {
	client       *http.Client
	token        *token
	netboxURL    *url.URL
	userAgent    string
	viewsInclude []string
	viewsExclude []string
}

// NewClient returns a new Netbox API client
func NewClient(token string, u *url.URL) (c *Client) {
	transport := &http.Transport{
		// connection pool
		MaxConnsPerHost:     100,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		// timeouts
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		// connection settings
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// enable http2
		ForceAttemptHTTP2: true,
		// large buffer
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 64 * 1024,
		// compression
		DisableCompression: false,
	}
	c = &Client{
		client: &http.Client{
			Timeout: defaultHTTPClientTimeout,
			Transport: &instrumentedTransport{
				RoundTripper:     transport,
				requestsDuration: newRequestDurationMetric(),
				requestsTotal:    newRequestsTotalMetric(),
			},
		},
		netboxURL: u,
		token:     newToken(token),
		userAgent: defaultUserAgent,
	}
	return
}

// Do executes an HTTP request against the Netbox API.
// `Authorization` and `User-Agent` headers are set automatically. Existing
// values are not modified.
func (c *Client) Do(r *http.Request) (
	resp *http.Response,
	err error,
) {
	r.Header.Set("Authorization", fmt.Sprintf("Token %s", c.token.raw))
	r.Header.Set("User-Agent", c.userAgent)
	r.Header.Set("Accept-Encoding", "gzip")
	resp, err = c.client.Do(r)
	return
}

// SetTLSConfig sets the TLS configuration to be used by the client.
func (c *Client) SetTLSConfig(t *tls.Config) {
	ct := c.client.Transport.(*instrumentedTransport)
	rt := ct.RoundTripper.(*http.Transport)
	rt.TLSClientConfig = t
}

// SetUserAgent sets the User-Agent header to be used by the client.
func (c *Client) SetUserAgent(useragent string) {
	c.userAgent = useragent
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
func (it *instrumentedTransport) RoundTrip(r *http.Request) (
	resp *http.Response,
	err error,
) {
	label := it.endpointLabel(r.URL.Path)
	metricStart := time.Now()
	resp, err = it.RoundTripper.RoundTrip(r)
	it.requestsDuration.observe(label, time.Since(metricStart).Seconds())
	if err != nil {
		it.requestsTotal.incError(label)
	}
	if resp != nil {
		it.requestsTotal.incStatusCode(label, resp.StatusCode)
	}
	return
}

func (*instrumentedTransport) endpointLabel(uriPath string) string {
	if !strings.HasPrefix(uriPath, fmt.Sprintf("/%s", apiPathString)) {
		return metricsFallbackEndpointLabel
	}
	remaining := strings.TrimPrefix(uriPath, fmt.Sprintf("/%s", apiPathString))
	if i := strings.IndexByte(remaining, '/'); i >= 0 {
		remaining = remaining[:i]
	}
	if remaining == "" {
		return metricsFallbackEndpointLabel
	}
	return remaining
}
