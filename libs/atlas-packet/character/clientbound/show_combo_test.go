package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterShowCombo version=gms_v83 ida=0x9602cb
// packet-audit:verify packet=character/clientbound/CharacterShowCombo version=gms_v84 ida=0x99f31e
// packet-audit:verify packet=character/clientbound/CharacterShowCombo version=gms_v87 ida=0x9e3794
// packet-audit:verify packet=character/clientbound/CharacterShowCombo version=gms_v92 ida=0x8fe8e0
// packet-audit:verify packet=character/clientbound/CharacterShowCombo version=gms_v95 ida=0x91a970
// packet-audit:verify packet=character/clientbound/CharacterShowCombo version=jms_v185 ida=0xa2d40d
//
// NOTE: these addresses are CUserLocal::OnIncComboResponse's own function
// entry (the handler that does the Decode4 + DrawCombo), not the OnPacket
// dispatcher's jump-table call site. design.md §2.2 cites the dispatcher
// call-site address per version (0x95086b/0x988698/0x9cbcd4/0x913a68/
// 0x934238/0xa1208d) to document the opcode-to-handler binding; the marker
// here must match the address the evidence record and audit report resolve
// via the registry fname (CUserLocal::OnIncComboResponse), which is the
// handler's own entry (task-217, VERIFYING_A_PACKET.md "orphan marker"
// failure mode: fix the address to match, never weaken the check).
func TestShowComboRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowCombo(37)
			output := ShowCombo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Count() != 37 {
				t.Errorf("count round-trip mismatch: want 37, got %d", output.Count())
			}
		})
	}
}

// TestShowComboByteFixture pins CUserLocal::OnIncComboResponse's wire: a
// single Decode4 into m_nCombo, then DrawCombo (design.md §2.2). No version
// divergence -- the body is identical on all six in-scope versions.
func TestShowComboByteFixture(t *testing.T) {
	cases := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"gms_v83", "GMS", 83, 1},
		{"gms_v84", "GMS", 84, 1},
		{"gms_v87", "GMS", 87, 1},
		{"gms_v92", "GMS", 92, 1},
		{"gms_v95", "GMS", 95, 1},
		{"jms_v185", "JMS", 185, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.major, c.minor)
			got := pt.Encode(t, ctx, NewShowCombo(1).Encode, nil)
			want := []byte{0x01, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("length mismatch: want % x, got % x", want, got)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte mismatch: want % x, got % x", want, got)
				}
			}
		})
	}
}
