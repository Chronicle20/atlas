package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/CharacterAranComboCounterRequest version=gms_v83 ida=0x9602f3
// packet-audit:verify packet=character/serverbound/CharacterAranComboCounterRequest version=gms_v84 ida=0x99f346
// packet-audit:verify packet=character/serverbound/CharacterAranComboCounterRequest version=gms_v87 ida=0x9e37bc
// packet-audit:verify packet=character/serverbound/CharacterAranComboCounterRequest version=gms_v92 ida=0x8ef840
// packet-audit:verify packet=character/serverbound/CharacterAranComboCounterRequest version=gms_v95 ida=0x909070
// packet-audit:verify packet=character/serverbound/CharacterAranComboCounterRequest version=jms_v185 ida=0xa2d435
func TestAranComboCounterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AranComboCounterRequest{}
			output := AranComboCounterRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestAranComboCounterEmptyBody pins CUserLocal::RequestIncCombo's wire on
// every in-scope version: the function is an m_bHoldCombo guard plus
// COutPacket(op) plus SendPacket, with zero Encode calls (design.md §2.1).
func TestAranComboCounterEmptyBody(t *testing.T) {
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
			got := pt.Encode(t, ctx, AranComboCounterRequest{}.Encode, nil)
			if len(got) != 0 {
				t.Errorf("expected empty body, got % x", got)
			}
		})
	}
}
