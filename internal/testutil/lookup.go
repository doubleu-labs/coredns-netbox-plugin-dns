package testutil

import (
	"fmt"
	"slices"
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

type LookupTestCase struct {
	test.Case
	Name        string
	Rcode       int
	WantErr     bool
	WantSendErr bool
}

func RunLookupTests(t *testing.T, tcs []LookupTestCase, s *TestServer) {
	for tc := range slices.Values(tcs) {
		var name string
		if tc.Name != "" {
			name = tc.Name
		} else {
			name = fmt.Sprintf("%s %s", tc.Qname, dns.TypeToString[tc.Qtype])
		}
		t.Run(
			name,
			func(t *testing.T) {
				msg := tc.Msg()
				resp, sendErr := s.Send(msg)
				if tc.WantSendErr {
					if sendErr == nil {
						t.Fatal("expected send error, got none")
					}
					return
				}

				if sendErr != nil {
					t.Fatalf("failed to send message: %v", sendErr)
				}

				if resp == nil {
					t.Fatal("expected response, got none")
				}

				if resp.Rcode != tc.Rcode {
					t.Errorf(
						"unexpected rcode: got %s, want %s",
						dns.RcodeToString[resp.Rcode],
						dns.RcodeToString[tc.Rcode],
					)
				}

				if ok := RunLookupTestCheckCNAME(t, tc, resp); !ok {
					return
				}

				checkErr := test.SortAndCheck(resp, tc.Case)

				if tc.WantErr {
					if checkErr == nil {
						t.Error("expected check error, got none")
					}
					return
				}

				if checkErr != nil {
					t.Error(checkErr)
				}
			},
		)
	}
}

func RunLookupTestCheckCNAME(
	t *testing.T,
	tc LookupTestCase,
	resp *dns.Msg,
) bool {
	if err := test.CNAMEOrder(resp); err != nil {
		t.Errorf("cname response out of order")
		return false
	}
	if tc.Qtype == dns.TypeCNAME || RunTestLookupContainsCNAME(resp) {
		if err := test.Header(tc.Case, resp); err != nil {
			t.Error(err)
		}
		if err := test.Section(tc.Case, test.Answer, resp.Answer); err != nil {
			t.Error(err)
		}
		if err := test.Section(tc.Case, test.Ns, resp.Ns); err != nil {
			t.Error(err)
		}
		if err := test.Section(tc.Case, test.Extra, resp.Extra); err != nil {
			t.Error(err)
		}
		return false
	}
	return true
}

func RunTestLookupContainsCNAME(resp *dns.Msg) bool {
	var out bool
	for answerRR := range slices.Values(resp.Answer) {
		if answerRR.Header().Rrtype == dns.TypeCNAME {
			out = true
		}
	}
	return out
}
