package config

import (
	"fmt"
	"maps"
	"sync"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
)

// Token is the shape of a configuration token.
type Token[T Config] interface {
	Parse(*caddy.Controller, T) error
	Validate(*caddy.Controller, log.P, T) error
}

type tokenMapType[T Config] map[string]Token[T]

// TokenMap associates a token name with a token struct.
type TokenMap[T Config] struct {
	tokenMapType[T]
}

// Process iterates registered tokens and calls their respective Parse and
// Validate methods.
func (tm TokenMap[T]) process(c *caddy.Controller, l log.P, cfg *T) error {
	for c.NextBlock() {
		tokenName := c.Val()
		token, ok := tm.tokenMapType[tokenName]
		if !ok {
			return tm.unknown(c, tokenName)
		}
		if err := token.Parse(c, *cfg); err != nil {
			return err
		}
	}

	for tokenName, token := range tm.tokenMapType {
		if err := token.Validate(c, l, *cfg); err != nil {
			return c.Err(
				core.ScopedMessage(
					"config",
					fmt.Sprintf(
						"error validating token %q; %v",
						tokenName,
						err.Error(),
					),
				),
			)
		}
	}

	return nil
}

func (tm TokenMap[T]) tokens() (out string) {
	var i int
	for token := range maps.Keys(tm.tokenMapType) {
		out += fmt.Sprintf("%q", token)
		if i+1 < len(tm.tokenMapType) {
			out += ", "
		}
		if i == len(tm.tokenMapType)-2 {
			out += "or "
		}
		i++
	}
	return
}

func (tm TokenMap[T]) unknown(c *caddy.Controller, token string) error {
	return c.Err(
		core.ScopedMessage(
			"config",
			fmt.Sprintf("unknown token %q; expected %s", token, tm.tokens()),
		),
	)
}

// RegisterToken registers a token in the specified TokenMap.
func RegisterToken[T Config](
	so *sync.Once,
	tm *TokenMap[T],
	tokenName string,
	t Token[T],
) {
	so.Do(
		func() {
			if tm == nil {
				tm = new(TokenMap[T])
			}
			if tm.tokenMapType == nil {
				tm.tokenMapType = make(tokenMapType[T])
			}
		},
	)
	tm.tokenMapType[tokenName] = t
}

// ErrNoTokenValue returns an error indicating that a value is required for a
// token but was not provided.
func ErrNoTokenValue(c *caddy.Controller, tokenName string) error {
	return c.Err(
		core.ScopedMessage(
			"config",
			fmt.Sprintf("no value for token %q provided", tokenName),
		),
	)
}

// ErrTokenParse returns an error indicating that a token value could not be
// parsed.
func ErrTokenParse(c *caddy.Controller, tokenName string, err error) error {
	return c.Err(
		core.ScopedMessage(
			"config",
			fmt.Sprintf("error parsing token %q: %v", tokenName, err),
		),
	)
}
