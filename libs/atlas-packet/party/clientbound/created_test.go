package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v95 ida=0xa10efc
// packet-audit:verify packet=party/clientbound/PartyCreated version=jms_v185 ida=0xb297e7
func TestCreatedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCreated(7, 12345)
			output := Created{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
		})
	}
}
