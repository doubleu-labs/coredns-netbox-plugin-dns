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
	activeStatus []string
	viewsInclude []string
	viewsExclude []string
}

// NewClient returns a new Netbox API client
func NewClient(token string, u *url.URL) *Client {
	return &Client{
		client: &http.Client{
			Timeout: defaultHTTPClientTimeout,
			Transport: &instrumentedTransport{
				RoundTripper: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 100,
					IdleConnTimeout:     90 * time.Second,
					DialContext: (&net.Dialer{
						Timeout:   500 * time.Millisecond,
						KeepAlive: 30 * time.Second,
					}).DialContext,
					TLSHandshakeTimeout:   500 * time.Millisecond,
					ResponseHeaderTimeout: 500 * time.Millisecond,
				},
				requestsDuration: newRequestDurationMetric(),
				requestsTotal:    newRequestsTotalMetric(),
			},
		},
		netboxURL: u,
		token:     newToken(token),
		userAgent: defaultUserAgent,
	}
}

// Do executes an HTTP request against the Netbox API.
// `Authorization` and `User-Agent` headers are set automatically. Existing
// values are not modified.
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	r.Header.Set(
		"Authorization",
		fmt.Sprintf("Token %s", c.token.raw),
	)
	r.Header.Set("User-Agent", c.userAgent)
	return c.client.Do(r)
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
	*http.Response,
	error,
) {
	label := it.endpointLabel(r.URL.Path)
	metricStart := time.Now()
	response, err := it.RoundTripper.RoundTrip(r)
	it.requestsDuration.observe(label, time.Since(metricStart).Seconds())
	if err != nil {
		it.requestsTotal.incError(label)
		return response, err
	}
	it.requestsTotal.incStatusCode(label, response.StatusCode)
	return response, nil
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
