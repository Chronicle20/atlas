package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=gms_v95 ida=0xa1169e
// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=jms_v185 ida=0xb297e7
func TestChangeLeaderRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeLeader(14, 9999, true)
			output := ChangeLeader{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
			if output.Disconnected() != input.Disconnected() {
				t.Errorf("disconnected: got %v, want %v", output.Disconnected(), input.Disconnected())
			}
		})
	}
}
