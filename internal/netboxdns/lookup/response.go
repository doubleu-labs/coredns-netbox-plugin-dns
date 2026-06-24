package lookup

import "github.com/miekg/dns"

// Result is the state of a Lookup response.
type Result int

const (
	// Success indicates that the Lookup was successful.
	Success Result = iota

	// NameError indicates that the name does not exist or is not found in any
	// of the configured views.
	NameError

	// Delegation indicates that the name is a delegated zone.
	Delegation
)

// Response is the result of a Lookup.
type Response struct {
	Answer []dns.RR
	Ns     []dns.RR
	Extra  []dns.RR
	Result Result
}
