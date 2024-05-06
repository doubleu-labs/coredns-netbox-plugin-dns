package netboxdns

import (
	"testing"

	"github.com/coredns/caddy"
)

type SetupTest struct {
	Name     string
	Corefile string
	WantErr  bool
}

var setupTests []SetupTest = []SetupTest{
	{
		"no configuration",
		`netboxdns`,
		true,
	},
	{
		"unknown token",
		`netboxdns {
			noop
		}`,
		true,
	},
	{
		"no netbox token specified",
		`netboxdns {
			url http://localhost:9999/
		}`,
		true,
	},
	{
		"no value for netbox token",
		`netboxdns {
			url http://localhost:9999/
			token
		}`,
		true,
	},
	{
		"no netbox url specified",
		`netboxdns {
			token sometoken
		}`,
		true,
	},
	{
		"no value for netbox url",
		`netboxdns {
			token sometoken
			url
		}`,
		true,
	},
	{
		"minimum valid configuration",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
		}`,
		false,
	},
	{
		"invalid netbox url value",
		`netboxdns {
			token sometoken
			url "http://local host:9999/"
		}`,
		true,
	},
	{
		"multiple configurations",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
		}
		netboxdns {
			token sometoken
			url http://localhost:9999/
		}`,
		true,
	},
	{
		"configuration with responsible zone",
		`netboxdns example.com {
			token sometoken
			url http://localhost:9999/
		}`,
		false,
	},
	{
		"no value for timeout specified",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			timeout
		}`,
		true,
	},
	{
		"minimum configuration with timeout",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			timeout 10s
		}`,
		false,
	},
	{
		"invalid timeout",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			timeout 10g
		}`,
		true,
	},
	{
		"minimum configuration fallthrough all zones",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			fallthrough
		}`,
		false,
	},
	{
		"minimum configuration fallthrough specified zone",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			fallthrough example.net
		}`,
		false,
	},
	{
		"minimum configuration tls system ca",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			tls
		}`,
		false,
	},
	{
		"minimum configuration tls nonexistant file",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			tls noop.pem
		}`,
		true,
	},
	{
		"minimum configuration tls private ca",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			tls .testing/tls/ca.pem
		}`,
		false,
	},
	{
		"minimum configuration tls client auth",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			tls .testing/tls/client.pem .testing/tls/client-key.pem
		}`,
		false,
	},
	{
		"minimum configuration tls client auth private ca",
		`netboxdns {
			token sometoken
			url http://localhost:9999/
			tls .testing/tls/client.pem .testing/tls/client-key.pem .testing/tls/ca.pem
		}`,
		false,
	},
}

func TestSetup(t *testing.T) {
	for _, tt := range setupTests {
		t.Run(tt.Name, func(t *testing.T) {
			controller := caddy.NewTestController("dns", tt.Corefile)
			if err := setup(controller); (err != nil) != tt.WantErr {
				t.Errorf(
					"setup error: %v, wanterr: %t",
					err,
					tt.WantErr,
				)
			}
		})
	}
}
