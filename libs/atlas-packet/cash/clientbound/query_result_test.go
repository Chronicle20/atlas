package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
