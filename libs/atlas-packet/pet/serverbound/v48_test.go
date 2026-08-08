package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPetSpawnBytesV48 pins the v48 pet-activate body, which is the v72 body
// MINUS the trailing lead byte.
//
// IDA v48 CWvsContext::SendActivatePetRequest @0x71d118 builds COutPacket(77) and
// encodes only Encode4(tick) then Encode2(slot) before CClientSocket::SendPacket.
// v72 @0x91c241 builds COutPacket(97) and encodes Encode4(tick) @0x91c513,
// Encode2(slot) @0x91c51e AND Encode1(lead) @0x91c529.
//
// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v48 ida=0x71d118
func TestPetSpawnBytesV48(t *testing.T) {
	in := Spawn{updateTime: 0x01020304, slot: 2, lead: true}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime — Encode4
		0x02, 0x00, // slot       — Encode2
		// no lead byte on v48.
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 pet spawn:\n got % x\nwant % x", got, want)
	}
}

// TestPetSpawnLeadBoundary guards the boundary: v48 stops after the slot short,
// v72 onward append the lead byte. v61 follows the v72 shape by inference only -
// see hasPetSpawnLead for why nothing depends on that choice.
func TestPetSpawnLeadBoundary(t *testing.T) {
	in := Spawn{updateTime: 0x01020304, slot: 2, lead: true}
	v48 := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	for _, v := range []uint16{72, 79, 83, 87, 95} {
		got := in.Encode(nil, pt.CreateContext("GMS", v, 1))(nil)
		want := append(append([]byte{}, v48...), 0x01)
		if !bytes.Equal(got, want) {
			t.Errorf("GMS v%d pet spawn:\n got % x\nwant % x", v, got, want)
		}
	}
}

// TestPetFoodBytesV48 — CWvsContext::SendPetFoodItemUseRequest @0x70df2f builds
// COutPacket(60) and encodes Encode4(tick), Encode2(slot), Encode4(itemId).
// Shape-stable; the codec already matched.
//
// packet-audit:verify packet=pet/serverbound/PetFood version=gms_v48 ida=0x70df2f
func TestPetFoodBytesV48(t *testing.T) {
	in := Food{updateTime: 0x01020304, source: 2, itemId: 2120000}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime — Encode4
		0x02, 0x00, // source     — Encode2
		0x40, 0x59, 0x20, 0x00, // itemId 2120000 (0x205940) — Encode4
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 pet food:\n got % x\nwant % x", got, want)
	}
}
