package api

import "fmt"

const redactedToken = "[REDACTED]"

type token struct {
	raw string
}

func newToken(s string) *token {
	return &token{
		raw: s,
	}
}

// Format implements fmt.Formatter
func (token) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(redactedToken))
}
