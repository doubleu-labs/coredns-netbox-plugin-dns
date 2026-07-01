package netboxdns

//noinspection LongLine
import (
	"fmt"
	"testing"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/testutil"
)

type testConfig struct {
	Fall         string
	NetboxURL    string
	NoOp         string
	Timeout      string
	TLS          string
	Token        string
	Views        string
	ViewsExclude string
	ViewPoller   string
	Zones        string
}

func newTestConfig() *testConfig {
	return &testConfig{}
}

func newValidTestConfig() *testConfig {
	return newTestConfig().
		set("url", "http://127.0.0.1:12345/").
		set("token", "a_token")
}

func (c *testConfig) set(k, v string) *testConfig {
	switch k {
	case "fallthrough":
		c.Fall = v
	case "url":
		c.NetboxURL = v
	case "noop":
		c.NoOp = v
	case "timeout":
		c.Timeout = v
	case "tls":
		c.TLS = v
	case "token":
		c.Token = v
	case "views":
		c.Views = v
	case "views_exclude":
		c.ViewsExclude = v
	case "view_poller":
		c.ViewPoller = v
	case "zones":
		c.Zones = v
	default:
		break
	}
	return c
}

func (c *testConfig) name() string {
	return "netboxdns"
}

func (c *testConfig) format() string {
	out := fmt.Sprintf("%s ", c.name())
	if len(c.Zones) != 0 {
		out += fmt.Sprintf("%s ", c.Zones)
	}
	out += "{\n"

	if str, ok := c.formatTokenWithArguments("fallthrough", c.Fall); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("url", c.NetboxURL); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("noop", c.NoOp); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("timeout", c.Timeout); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("tls", c.TLS); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("token", c.Token); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("views", c.Views); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments(
		"views_exclude",
		c.ViewsExclude,
	); ok {
		out += str
	}

	if str, ok := c.formatTokenWithArguments("view_poller", c.ViewPoller); ok {
		out += str
	}

	out += "}"
	return out
}

func (c *testConfig) formatTokenWithArguments(
	token string,
	value string,
) (string, bool) {
	var out string
	if value == "{TRUE}" {
		out = fmt.Sprintf("\t%s\n", token)
		return out, true
	} else if value != "" {
		out = fmt.Sprintf("\t%s %s\n", token, value)
		return out, true
	}
	return out, false
}

func Test_SetupBasic(t *testing.T) {
	tt := []testutil.SetupTest{
		{
			Name:        "no configuration body",
			ServerBlock: `netboxdns`,
			WantErr:     true,
		},
		{
			Name:        "empty configuration body same line",
			ServerBlock: `netboxdns {}`,
			WantErr:     true,
		},
		{
			Name: "empty configuration body new line",
			ServerBlock: `netboxdns {
		}`,
			WantErr: true,
		},
		{
			Name: "unknown token",
			ServerBlock: `netboxdns {
			not_a_token
		}`,
			WantErr: true,
		},
		{
			Name: "multiple plugins",
			ServerBlock: fmt.Sprintf(
				"%s\n%s\n",
				newValidTestConfig().format(),
				newValidTestConfig().format(),
			),
			WantErr: true,
		},
		{
			Name: "url not set",
			ServerBlock: newValidTestConfig().
				set("url", "{TRUE}").
				format(),
			WantErr: true,
		},
		{
			Name: "token not set",
			ServerBlock: newValidTestConfig().
				set("token", "{TRUE}").
				format(),
			WantErr: true,
		},
		{
			Name: "url invalid",
			ServerBlock: newValidTestConfig().
				set("url", "'http://local host:12345'").
				format(),
			WantErr: true,
		},
		{
			Name:        "minimum valid configuration",
			ServerBlock: newValidTestConfig().format(),
		},
		{
			Name: "enable fallthrough",
			ServerBlock: newValidTestConfig().
				set("fallthrough", "{TRUE}").
				format(),
		},
		{
			Name: "noop enabled",
			ServerBlock: newValidTestConfig().
				set("noop", "{TRUE}").
				format(),
		},
		{
			Name: "timeout no value",
			ServerBlock: newValidTestConfig().
				set("timeout", "{TRUE}").
				format(),
			WantErr: true,
		},
		{
			Name: "timeout invalid value",
			ServerBlock: newValidTestConfig().
				set("timeout", "invalid").
				format(),
			WantErr: true,
		},
		{
			Name: "timeout valid value",
			ServerBlock: newValidTestConfig().
				set("timeout", "10s").
				format(),
		},
		{
			Name: "tls no value",
			ServerBlock: newValidTestConfig().
				set("tls", "{TRUE}").
				format(),
			WantErr: true,
		},
		{
			Name: "tls too many args",
			ServerBlock: newValidTestConfig().
				set("tls", "arg_one arg_two arg_three arg_four").
				format(),
			WantErr: true,
		},
		{
			Name: "tls ca cert",
			ServerBlock: newValidTestConfig().
				set("tls", "../../../testdata/ca.pem").
				format(),
		},
		{
			Name: "tls cert and key",
			ServerBlock: newValidTestConfig().
				set(
					"tls",
					"../../../testdata/client.pem "+
						"../../../testdata/client-key.pem",
				).
				format(),
		},
		{
			Name: "tls cert, key and ca cert",
			ServerBlock: newValidTestConfig().
				set(
					"tls",
					"../../../testdata/client.pem "+
						"../../../testdata/client-key.pem "+
						"../../../testdata/ca.pem",
				).
				format(),
		},
		{
			Name: "views no value",
			ServerBlock: newValidTestConfig().
				set("views", "{TRUE}").
				format(),
			WantErr: true,
		},
		{
			Name: "one view",
			ServerBlock: newValidTestConfig().
				set("views", "view_one").
				format(),
		},
		{
			Name: "multiple views",
			ServerBlock: newValidTestConfig().
				set("views", "view_one view_two").
				format(),
		},
		{
			Name: "views multiple declarations",
			ServerBlock: `netboxdns {
				token a_token
				url http://127.0.0.1:12345/
				views view_one
				views view_two
			}`,
		},
		{
			Name: "views exclude no value",
			ServerBlock: newValidTestConfig().
				set("views_exclude", "{TRUE}").
				format(),
			WantErr: true,
		},
		{
			Name: "one view exclude",
			ServerBlock: newValidTestConfig().
				set("views_exclude", "view_one").
				format(),
		},
		{
			Name: "multiple views exclude",
			ServerBlock: newValidTestConfig().
				set("views_exclude", "view_one view_two").
				format(),
		},
		{
			Name: "views exclude multiple declarations",
			ServerBlock: `netboxdns {
				token a_token
				url http://127.0.0.1:12345/
				views_exclude view_one
				views_exclude view_two
			}`,
		},
		{
			Name: "view poller no value",
			ServerBlock: newValidTestConfig().
				set("view_poller", "{TRUE}").
				format(),
		},
		{
			Name: "view poller invalid duration",
			ServerBlock: newValidTestConfig().
				set("view_poller", "invalid").
				format(),
			WantErr: true,
		},
		{
			Name: "view poller valid duration",
			ServerBlock: newValidTestConfig().
				set("view_poller", "10s").
				format(),
		},
	}

	testutil.RunSetupTests(t, tt, setup, &isSetupTest)
}
