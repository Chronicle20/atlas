package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v95 ida=0xa10ab0
func TestInviteRejectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InviteReject{unk: 1, from: "SomePartyLeader"}
			output := InviteReject{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
			if output.From() != input.From() {
				t.Errorf("from: got %v, want %v", output.From(), input.From())
			}
		})
	}
}
