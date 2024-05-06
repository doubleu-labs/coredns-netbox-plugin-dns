package netboxdns

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/netbox"
	"github.com/miekg/dns"
)

var txtMultiValueRegexp *regexp.Regexp

func init() {
	txtMultiValueRegexp = regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)`)
}

func recordsToRR(records []netbox.Record) ([]dns.RR, error) {
	out := make([]dns.RR, 0, len(records))
	for _, record := range records {
		qtype := dns.StringToType[record.Type]
		switch qtype {
		case dns.TypeTXT:
			out = append(out, recordToTXT(record))
		default:
			rrStr := fmt.Sprintf(
				"%s %d IN %s %s",
				record.FQDN,
				*record.TTL,
				record.Type,
				record.Value,
			)
			rr, err := dns.NewRR(rrStr)
			if err != nil {
				return out, err
			}
			out = append(out, rr)
		}
	}
	return out, nil
}

func recordToTXT(record netbox.Record) *dns.TXT {
	txt := make([]string, 0)
	if strings.HasPrefix(record.Value, `"`) {
		values := txtMultiValueRegexp.FindAllString(record.Value, -1)
		for i := range values {
			values[i] = strings.Trim(values[i], `"`)
			values[i] = strings.ReplaceAll(values[i], "\\r\\n", "")
			values[i] = strings.ReplaceAll(values[i], "\\n", "")
			values[i] = strings.TrimSpace(values[i])
			if values[i] != "" {
				txt = append(txt, values[i])
			}
		}
	} else {
		txt = append(txt, record.Value)
	}
	return &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   record.FQDN,
			Ttl:    *record.TTL,
			Class:  dns.ClassINET,
			Rrtype: dns.TypeTXT,
		},
		Txt: txt,
	}
}

func filterRRByType(rrs []dns.RR, recordType uint16) []dns.RR {
	out := make([]dns.RR, 0)
	for _, rr := range rrs {
		if rr.Header().Rrtype == recordType {
			out = append(out, rr)
		}
	}
	return out
}
