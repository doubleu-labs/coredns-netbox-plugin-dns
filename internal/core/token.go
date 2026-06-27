package core

import (
	"fmt"
	"maps"
	"sync"

	"github.com/coredns/caddy"
)

// Token is the shape of a configuration token.
type Token[T any] interface {
	Parse(*caddy.Controller, *T) error
	Validate(*caddy.Controller, *T) error
}

type tokenMapType[T any] map[string]Token[T]

// TokenMap associates a token name with a token struct.
type TokenMap[T any] struct {
	tokenMapType[T]
	PluginName string
}

func (tm TokenMap[T]) tokens() string {
	var out string
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
	return out
}

func (tm TokenMap[T]) unknown(c *caddy.Controller, token string) error {
	return c.Err(
		ScopedMessage(
			tm.PluginName,
			"config",
			fmt.Sprintf("unknown token %q; expected %s", token, tm.tokens()),
		),
	)
}

func tokenMapParse[T any](c *caddy.Controller, tm TokenMap[T], cfg *T) error {
	for c.NextBlock() {
		tokenName := c.Val()
		token, ok := tm.tokenMapType[tokenName]
		if !ok {
			return tm.unknown(c, tokenName)
		}
		if err := token.Parse(c, cfg); err != nil {
			return err
		}
	}
	return nil
}

func tokenMapValidate[T any](
	c *caddy.Controller,
	tm TokenMap[T],
	cfg *T,
) error {
	for tokenName, token := range tm.tokenMapType {
		if err := token.Validate(c, cfg); err != nil {
			return c.Err(
				ScopedMessage(
					tm.PluginName,
					"config",
					fmt.Sprintf(
						"error validating token %q; %v",
						tokenName,
						err,
					),
				),
			)
		}
	}
	return nil
}

// ProcessTokens iterates the provided TokenMap and runs the Parse and Validate
// methods. Will return an error if a token is not registered in the TokenMap.
// Will return an error if a token-associated configuration field is marked as
// required and is missing, empty, or zero.
func ProcessTokens[T any](
	c *caddy.Controller,
	tm TokenMap[T],
	cfg *T,
) error {
	if err := tokenMapParse(c, tm, cfg); err != nil {
		return err
	}
	if err := tokenMapValidate(c, tm, cfg); err != nil {
		return err
	}
	return nil
}

// RegisterToken registers a token in the specified TokenMap.
func RegisterToken[T any](
	so *sync.Once,
	tm *TokenMap[T],
	tokenName string,
	t Token[T],
) {
	so.Do(
		func() {
			if tm == nil {
				tm = new(TokenMap[T])
				tm.tokenMapType = make(tokenMapType[T])
			}
		},
	)
	tm.tokenMapType[tokenName] = t
}

// ErrNoTokenValue returns an error indicating that a value is required for a
// token but was not provided.
func ErrNoTokenValue(c *caddy.Controller, pluginName, tokenName string) error {
	return c.Err(
		ScopedMessage(
			pluginName,
			"config",
			fmt.Sprintf("no value for token %q provided", tokenName),
		),
	)
}

// ErrTokenParse returns an error indicating that a token value could not be
// parsed.
func ErrTokenParse(
	c *caddy.Controller,
	pluginName, tokenName string,
	err error,
) error {
	return c.Err(
		ScopedMessage(
			pluginName,
			"config",
			fmt.Sprintf("error parsing token %q: %v", tokenName, err),
		),
	)
}
