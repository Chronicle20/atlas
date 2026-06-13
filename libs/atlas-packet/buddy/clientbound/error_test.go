package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/clientbound/BuddyError version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyError version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyError version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyError version=jms_v185 ida=0xb2a873
func TestBuddyErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyError(10, false)
			output := Error{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

func TestBuddyErrorWithExtraRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyError(11, true)
			output := Error{hasExtra: true}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
