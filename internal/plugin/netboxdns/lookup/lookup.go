package lookup

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/api"
	"github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/core"
	"github.com/miekg/dns"
)

// Lookup contains the parameters of a request that will be used to lookup
// records in Netbox.
type Lookup struct {
	Client *api.Client
	Family int
	Logger *log.P
	QName  string
	QType  uint16
	Views  *core.Views
}

func (l *Lookup) debug(msg string) {
	l.Logger.Debug(core.ScopedMessage("lookup", msg))
}

// Run the lookup.
func (l *Lookup) Run() (*Response, error) {
	nameTrimmed := strings.TrimSuffix(l.QName, ".")

	zone, err := l.matchZone(nameTrimmed)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		l.debug(fmt.Sprintf("no zone matching %q", l.QName))
		response := &Response{
			Result: NameError,
		}
		return response, nil
	}
	if _, ok := dns.TypeToString[l.QType]; !ok {
		zoneRRs, zoneRRsErr := zone.ToRR()
		if zoneRRsErr != nil {
			return nil, zoneRRsErr
		}
		resp := &Response{
			Ns:     zoneRRs,
			Result: Success,
		}
		return resp, nil
	}

	if nameTrimmed == zone.Name {
		var originError error
		origin, originError := l.processOrigin(zone)
		if originError != nil {
			return nil, err
		}
		if origin != nil {
			l.debug(
				fmt.Sprintf(
					"found origin records for [%s] %q",
					dns.TypeToString[l.QType],
					l.QName,
				),
			)
			return origin, nil
		}
	}

	explicitResponse, err := l.explicit(zone)
	if err != nil {
		return nil, err
	}
	if explicitResponse != nil {
		l.debug(
			fmt.Sprintf(
				"found records for [%s] %q",
				dns.TypeToString[l.QType],
				l.QName,
			),
		)
		return explicitResponse, nil
	}

	delegateResponse, err := l.delegate(zone, l.QName)
	if err != nil {
		return nil, err
	}
	if delegateResponse != nil {
		l.debug(fmt.Sprintf("found delegate records for %q", l.QName))
		return delegateResponse, nil
	}

	l.debug(
		fmt.Sprintf(
			"no records found for [%s] %q",
			dns.TypeToString[l.QType],
			l.QName,
		),
	)

	response := &Response{
		Result: NameError,
	}
	return response, nil
}

func (l *Lookup) explicit(z *api.Zone) (*Response, error) {
	recordQuery := &api.RecordQuery{
		FQDN: l.QName,
		Type: l.explicitQueryTypes(),
		Zone: z,
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	records, err := recordQuery.GetRecords(ctx, l.Client)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	answerRRs, err := records.ToRRs()
	if err != nil {
		return nil, err
	}

	extraRecords, err := l.processExtra(z, answerRRs)
	if err != nil {
		return nil, err
	}

	extraRRs, err := extraRecords.ToRRs()
	if err != nil {
		return nil, err
	}

	return l.explicitResponse(answerRRs, extraRRs), nil
}

func (l *Lookup) explicitQueryTypes() []string {
	queryType, ok := dns.TypeToString[l.QType]
	if !ok {
		return nil
	}
	queryTypes := []string{queryType}
	if l.QType == dns.TypeA || l.QType == dns.TypeAAAA {
		queryTypes = append(queryTypes, "CNAME")
	}
	return queryTypes
}

func (l *Lookup) explicitResponse(answerRRs, extraRRs []dns.RR) *Response {
	cnameRRs := l.filterRRByType(answerRRs, dns.TypeCNAME)
	if l.QType == dns.TypeCNAME || len(cnameRRs) > 0 {
		answerRRs = slices.Concat(answerRRs, extraRRs)
		extraRRs = nil
	}
	return &Response{
		Answer: answerRRs,
		Extra:  extraRRs,
	}
}

func (l *Lookup) delegate(z *api.Zone, fqdn string) (*Response, error) {
	recordQuery := &api.RecordQuery{
		FQDN: l.QName,
		Type: []string{"NS"},
		Zone: z,
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()
	nameserverRecords, err := recordQuery.GetRecords(ctx, l.Client)
	if err != nil {
		return nil, err
	}
	if len(nameserverRecords) <= 0 {
		return nil, nil
	}
	nameserverRRs, err := nameserverRecords.ToRRs()
	if err != nil {
		return nil, err
	}
	extraRecords, err := l.processExtra(z, nameserverRRs)
	if err != nil {
		return nil, err
	}
	var extraRRs []dns.RR
	for r := range slices.Values(extraRecords) {
		var rrErr error
		rr, rrErr := r.ToRR()
		if rrErr != nil {
			return nil, rrErr
		}
		extraRRs = append(extraRRs, rr)
	}
	response := &Response{
		Answer: nameserverRRs,
		Extra:  extraRRs,
		Result: Delegation,
	}
	return response, nil
}
