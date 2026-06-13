package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/clientbound/CashQueryResult version=gms_v95 ida=0x496400
// packet-audit:verify packet=cash/clientbound/CashQueryResult version=jms_v185 ida=0x48b3e8
// packet-audit:verify packet=cash/clientbound/CashQueryResult version=gms_v87 ida=0x484616
// packet-audit:verify packet=cash/clientbound/CashQueryResult version=gms_v83 ida=0x478f81
// packet-audit:verify packet=cash/clientbound/CashQueryResult version=gms_v84 ida=0x47c0b3
func TestQueryResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCashQueryResult(100, 200, 300)
			output := QueryResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Credit() != input.Credit() {
				t.Errorf("credit: got %v, want %v", output.Credit(), input.Credit())
			}
			if output.Points() != input.Points() {
				t.Errorf("points: got %v, want %v", output.Points(), input.Points())
			}
			if v.Region == "GMS" && v.MajorVersion > 12 {
				if output.Prepaid() != input.Prepaid() {
					t.Errorf("prepaid: got %v, want %v", output.Prepaid(), input.Prepaid())
				}
			}
		})
	}
}

func TestQueryResultRoundTripNoPrepaid(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := NewCashQueryResult(500, 600, 700)
	output := QueryResult{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Credit() != input.Credit() {
		t.Errorf("credit: got %v, want %v", output.Credit(), input.Credit())
	}
	if output.Points() != input.Points() {
		t.Errorf("points: got %v, want %v", output.Points(), input.Points())
	}
	if output.Prepaid() != 0 {
		t.Errorf("prepaid: got %v, want 0 (should not be written for JMS)", output.Prepaid())
	}
}
