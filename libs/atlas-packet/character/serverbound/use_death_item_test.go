package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// UseDeathItem — CUserLocal::RequestUpgradeTombEffect. Every version encodes the
// same three little-endian 4-byte fields and differs only in the opcode:
//
//	Encode4  itemId   — hard-coded 5510000 (0x541370) by the client
//	Encode4  x        — m_ptRevive.x
//	Encode4  y        — m_ptRevive.y
//
// IDA gms_v95 CUserLocal::RequestUpgradeTombEffect@0x908320 (op 58 = 0x03A).
//
// packet-audit:verify packet=character/serverbound/UseDeathItem version=gms_v72 ida=0x867654
//
// The remaining seven versions (v79/v83/v84/v87/v92/v95/jms_v185) carry the
// identical wire layout per the IDA addresses in the doc comment above, but
// their packet-audit:verify markers are deliberately deferred to their own
// verification batches (task-210) — an unlinked marker with no evidence
// record/audit report fails `packet-audit matrix --check` as an orphan
// marker (VERIFYING_A_PACKET.md §8). Each batch adds its own marker line
// alongside its evidence pin.
func TestUseDeathItemByteOutput(t *testing.T) {
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
	// itemId 5510000 = 0x00541370; x = 100 (0x64); y = -200 (0xFFFFFF38).
	want := []byte{
		0x70, 0x13, 0x54, 0x00,
		0x64, 0x00, 0x00, 0x00,
		0x38, 0xFF, 0xFF, 0xFF,
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			got := NewUseDeathItem(5510000, 100, -200).Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Errorf("%s UseDeathItem wire: got %x want %x", v.name, got, want)
			}
		})
	}
}

func TestUseDeathItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUseDeathItem(5510000, 1234, -5678)
			output := UseDeathItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
