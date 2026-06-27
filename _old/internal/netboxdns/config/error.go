package config

import "github.com/coredns/caddy"

func ErrNoTokenValue(c *caddy.Controller, t string) error {
	return c.Errf(
		"[config] no value for token %q provided",
		t,
	)
}

func ErrTokenParse(c *caddy.Controller, t string, err error) error {
	return c.Errf(
		"[config] error parsing token %q: %v",
		t,
		err,
	)
}

func ErrTokenValidate(c *caddy.Controller, t string, m string) error {
	return c.Errf(
		"[config] error validating token %q: %s",
		t,
		m,
	)
}
