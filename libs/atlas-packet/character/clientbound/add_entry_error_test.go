package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v83 ida=0x5fa26c
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v87 ida=0x631b28
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v95 ida=0x5dabcd
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v84 ida=0x60f268
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v79 ida=0x5ceb55
func TestAddCharacterErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AddCharacterError{code: 3}
			output := AddCharacterError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
