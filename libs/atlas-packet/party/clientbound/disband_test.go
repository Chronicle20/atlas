package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v95 ida=0xa11085
// packet-audit:verify packet=party/clientbound/PartyDisband version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v84 ida=0xa89cf3
func TestDisbandRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewDisband(11, 5000, 300)
			output := Disband{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
		})
	}
}
