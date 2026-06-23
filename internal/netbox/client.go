package netbox

import (
	"fmt"
	"net/http"
	"net/url"
)

// Client for interacting with the Netbox API.
type Client struct {
	*http.Client
	NetboxURL *url.URL
	token     string
	UserAgent string
}

// String returns a string representation of the Client with sensitive
// information redacted.
func (c *Client) String() string {
	var tokenValue string
	if c.HasToken() {
		tokenValue = "[REDACTED]"
	}
	return fmt.Sprintf(
		"Client{NetboxURL: %s, token: %s, UserAgent: %s, Client: %#v}",
		c.NetboxURL,
		tokenValue,
		c.UserAgent,
		c.Client,
	)
}

// SetToken sets the token for the Client.
func (c *Client) SetToken(token string) {
	c.token = token
}

// HasToken returns true if the Client has a token set.
func (c *Client) HasToken() bool {
	return c.token != ""
}
