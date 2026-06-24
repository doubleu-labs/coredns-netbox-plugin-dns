package netbox

import (
	"fmt"
	"net/http"
	"net/url"
)

type token struct {
	raw string
}

func newToken(s string) *token {
	return &token{raw: s}
}

const redacted = "[REDACTED]"

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
