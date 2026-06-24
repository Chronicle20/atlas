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
