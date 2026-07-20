package lookup

import "github.com/miekg/dns"

type result int

const (
	// Success indicates that the Lookup was successful.
	Success result = iota

	// NameError indicates that the name was not found.
	NameError

	// Delegation indicates that the name is a delegated zone.
	Delegation
)

// Response contains the result of a Lookup.
//
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Response struct {
	Answer []dns.RR
	Ns     []dns.RR
	Extra  []dns.RR
	Result result
}
