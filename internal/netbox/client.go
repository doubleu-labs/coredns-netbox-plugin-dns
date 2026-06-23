package netbox

import (
	"fmt"
	"net/http"
	"net/url"
)

// Token is a string that prints a redaction statement.
type Token struct {
	raw string
}

const redacted = "[REDACTED]"

// Format redacts the token when printed.
func (t *Token) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(redacted))
}

// Client for interacting with the Netbox API.
type Client struct {
	*http.Client
	NetboxURL *url.URL
	token     *Token
	UserAgent string
}

// SetToken sets the token for the Client.
func (c *Client) SetToken(token string) {
	c.token = &Token{raw: token}
}

// HasToken returns true if the Client has a token set.
func (c *Client) HasToken() bool {
	if c.token == nil {
		return false
	}
	return c.token.raw != ""
}
