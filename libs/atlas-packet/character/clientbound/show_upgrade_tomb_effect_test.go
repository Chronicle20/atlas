package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ShowUpgradeTombEffect — CUserRemote::OnShowUpgradeTombEffect. The handler
// itself reads three Decode4s (itemId, nPosX, nPosY); the leading Decode4
// characterId is consumed by CUserPool::OnUserRemotePacket before the opcode
// switch, exactly as for every other CUserRemote::On* op.
//
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v72 ida=0x88d0e4
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v79 ida=0x8d9fe6
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v83 ida=0x983e40
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v84 ida=0x9c4206
// packet-audit:verify packet=character/clientbound/ShowUpgradeTombEffect version=gms_v87 ida=0xa098f2
//
// The remaining three versions (v92/v95/jms_v185) carry the
// identical wire layout per the IDA addresses in the doc comment above, but
// their packet-audit:verify markers are deliberately deferred to their own
// verification batches (task-210) — an unlinked marker with no evidence
// record/audit report fails `packet-audit matrix --check` as an orphan
// marker (VERIFYING_A_PACKET.md §8). Each batch adds its own marker line
// alongside its evidence pin.
func TestShowUpgradeTombEffectByteOutput(t *testing.T) {
	variants := []struct {
		name   string
		region string
		major  uint16
	}{
		{"gms_v72", "GMS", 72},
		{"gms_v79", "GMS", 79},
		{"gms_v83", "GMS", 83},
		{"gms_v84", "GMS", 84},
		{"gms_v87", "GMS", 87},
		{"gms_v92", "GMS", 92},
		{"gms_v95", "GMS", 95},
		{"jms_v185", "JMS", 185},
	}
	// characterId 4096 (0x1000); itemId 5510000 (0x541370); x 100; y -200.
	want := []byte{
		0x00, 0x10, 0x00, 0x00,
		0x70, 0x13, 0x54, 0x00,
		0x64, 0x00, 0x00, 0x00,
		0x38, 0xFF, 0xFF, 0xFF,
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			got := NewShowUpgradeTombEffect(4096, 5510000, 100, -200).Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Errorf("%s ShowUpgradeTombEffect wire: got %x want %x", v.name, got, want)
			}
		})
	}
}

func TestShowUpgradeTombEffectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowUpgradeTombEffect(77, 5510000, 1234, -5678)
			output := ShowUpgradeTombEffect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
