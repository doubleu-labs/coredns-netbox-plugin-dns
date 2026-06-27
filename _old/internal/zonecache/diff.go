package zonecache

import (
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

// Diff computes the IXFR delta between an old and a new snapshot of a zone.
// Both inputs are treated as multisets of RRs (duplicates are preserved) keyed
// by a stable canonical form: lower-cased owner name, type, class, and the
// rdata portion of dns.RR.String(). TTL is intentionally part of the key, so a
// pure TTL change shows up as one removed + one added RR — which matches what
// secondaries expect over IXFR.
//
// SOA records, if present in either snapshot, are filtered out: Transfer()
// emits the synthesised SOAs separately around the delta.
func Diff(oldRRs, newRRs []dns.RR) (removed, added []dns.RR) {
	oldMap := indexRRs(oldRRs)
	newMap := indexRRs(newRRs)

	for k, oldList := range oldMap {
		newList := newMap[k]
		// Anything in old beyond what new still has → removed.
		for i := len(newList); i < len(oldList); i++ {
			removed = append(removed, oldList[i])
		}
	}
	for k, newList := range newMap {
		oldList := oldMap[k]
		for i := len(oldList); i < len(newList); i++ {
			added = append(added, newList[i])
		}
	}
	return removed, added
}

// indexRRs builds a multiset keyed by canonical form. The slice value
// preserves insertion order so removed/added slices stay deterministic when
// the same key appears multiple times.
func indexRRs(rrs []dns.RR) map[string][]dns.RR {
	out := make(map[string][]dns.RR, len(rrs))
	for _, rr := range rrs {
		if rr == nil {
			continue
		}
		if rr.Header().Rrtype == dns.TypeSOA {
			continue
		}
		k := canonicalKey(rr)
		out[k] = append(out[k], rr)
	}
	return out
}

// canonicalKey returns a stable string identifying an RR for diff purposes.
// Format: "<lower owner>|<type>|<class>|<ttl>|<rdata>". Using dns.RR.String()
// for rdata avoids us hand-formatting every record type.
func canonicalKey(rr dns.RR) string {
	hdr := rr.Header()
	full := rr.String()
	// dns.RR.String() = "<owner>\t<ttl>\t<class>\t<type>\t<rdata>". The rdata
	// portion is whatever follows the four header tab-separated fields.
	rdata := full
	tabs := 0
	for i := 0; i < len(full); i++ {
		if full[i] == '\t' {
			tabs++
			if tabs == 4 {
				rdata = full[i+1:]
				break
			}
		}
	}
	var b strings.Builder
	b.Grow(len(hdr.Name) + len(rdata) + 16)
	b.WriteString(strings.ToLower(hdr.Name))
	b.WriteByte('|')
	b.WriteString(dns.TypeToString[hdr.Rrtype])
	b.WriteByte('|')
	b.WriteString(dns.ClassToString[hdr.Class])
	b.WriteByte('|')
	// TTL is part of the key so a pure TTL change shows up as one removed
	// + one added RR (which is what secondaries expect over IXFR).
	b.WriteString(strconv.FormatUint(uint64(hdr.Ttl), 10))
	b.WriteByte('|')
	b.WriteString(rdata)
	return b.String()
}
