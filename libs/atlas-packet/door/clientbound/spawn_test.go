package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestSpawnDoor pins the full wire body of spawnDoor (SPAWN_DOOR clientbound).
//
// the spawnDoor:
//
//	p.writeBool(launched) — 1 byte
//	p.writeInt(ownerid) — 4 bytes LE uint32
//	p.writePos(pos) — writeShort(x), writeShort(y) — 4 bytes
//
// Total: 9 bytes. Unbranched across all versions (no structural delta known).
//
// IDA gms_v83: CTownPortalPool::OnTownPortalCreated (0x7bd6c6, dispatched from
// CTownPortalPool::OnPacket case 0x113) reads Decode1(launched) → Decode4(ownerId)
// → Decode2(x) → Decode2(y) — byte-for-byte the layout pinned below.
//
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v83 ida=0x7bd6c6
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v84 ida=0x7e3740
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v87 ida=0x810af2
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v95 ida=0x762c00
// packet-audit:verify packet=door/clientbound/SpawnDoor version=jms_v185 ida=0x840fc6
// TestSpawnDoorV79 pins the gms_v79 SPAWN_DOOR (op 0x0FD/253) clientbound wire.
//
// IDA-verified client decode (GMS_v79_1_DEVM.exe, port 13340) —
// CTownPortalPool::OnTownPortalCreated @0x731176:
//
//	Decode1  @0x7311b6 → launched (bool, v3).
//	Decode4  @0x7311be → ownerId (key, v117).
//	Decode2  @0x731677 → x  (v111[1].Mid32, "new" branch; @0x73157f in "found" branch).
//	Decode2  @0x731687 → y  (v111[2].Lo32,  "new" branch; @0x731598 in "found" branch).
//
// Byte-for-byte identical to the v83 layout. atlas SpawnDoor.Encode writes
// WriteBool(launched) + WriteInt(ownerId) + WriteInt16(x) + WriteInt16(y) = 9 bytes.
//
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v79 ida=0x731176
//
// TestSpawnDoorV72 pins the gms_v72 SPAWN_DOOR (op 245) clientbound wire.
//
// IDA-verified client decode (GMS_v72.1_U_DEVM.exe, port 13339) —
// CTownPortalPool::OnTownPortalCreated @0x6f96a2:
//
//	Decode1 @0x6f96e2 → launched (bool, v3).
//	Decode4 @0x6f96ea → ownerId (key, v117).
//	Decode2 @0x6f9aa9 → x (found branch; @0x6f9ba1 in new branch).
//	Decode2 @0x6f9ac2 → y (found branch; @0x6f9bb1 in new branch).
//
// Byte-for-byte identical to the v79/v83 layout. 9 bytes.
//
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v72 ida=0x6f96a2
func TestSpawnDoorV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	m := NewSpawnDoor(1000, 100, 200, true)
	want := []byte{
		0x01,                   // Decode1 launched = true
		0xE8, 0x03, 0x00, 0x00, // Decode4 ownerId = 1000 LE
		0x64, 0x00, // Decode2 x = 100 LE
		0xC8, 0x00, // Decode2 y = 200 LE
	}
	if got := m.Encode(l, ctx)(nil); !bytes.Equal(got, want) {
		t.Errorf("v72 SpawnDoor golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestSpawnDoorV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	m := NewSpawnDoor(1000, 100, 200, true)
	want := []byte{
		0x01,                   // Decode1 launched = true
		0xE8, 0x03, 0x00, 0x00, // Decode4 ownerId = 1000 LE
		0x64, 0x00, // Decode2 x = 100 LE
		0xC8, 0x00, // Decode2 y = 200 LE
	}
	if got := m.Encode(l, ctx)(nil); !bytes.Equal(got, want) {
		t.Errorf("v79 SpawnDoor golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestSpawnDoor(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	// Input values: ownerId=1000, x=100, y=200, launched=true
	m := NewSpawnDoor(1000, 100, 200, true)

	// Golden wire layout (little-endian):
	// writeBool(true) → 0x01
	// writeInt(1000) → 0xE8 0x03 0x00 0x00
	// writeShort(100) → 0x64 0x00
	// writeShort(200) → 0xC8 0x00
	want := []byte{
		0x01,                   // launched = true
		0xE8, 0x03, 0x00, 0x00, // ownerid = 1000 LE
		0x64, 0x00, // x = 100 LE short
		0xC8, 0x00, // y = 200 LE short
	}

	// v83 golden bytes
	v83ctx := pt.CreateContext("GMS", 83, 1)
	v83 := m.Encode(l, v83ctx)(nil)
	if !bytes.Equal(v83, want) {
		t.Errorf("SpawnDoor v83 golden bytes mismatch\n got: % x\nwant: % x", v83, want)
	}

	// Cross-version equality: all known versions must produce identical bytes
	// (no structural branch implemented — single the v83 client layout applies to all).
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := m.Encode(l, ctx)(nil)
			if !bytes.Equal(got, v83) {
				t.Errorf("SpawnDoor %s differs from v83\n got: % x\nv83: % x", v.Name, got, v83)
			}
		})
	}
}
