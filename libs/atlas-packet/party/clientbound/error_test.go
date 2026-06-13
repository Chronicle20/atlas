package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/clientbound/PartyError version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyError version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyError version=gms_v95 ida=0xa10e9e
// packet-audit:verify packet=party/clientbound/PartyError version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyError version=gms_v84 ida=0xa89cf3
func TestErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewError(15, "SomePlayer")
			output := Error{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
