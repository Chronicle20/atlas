package clientbound

import (
	"bytes"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestSpawnPortal pins the full wire body of spawnPortal (SPAWN_PORTAL clientbound).
//
// Cosmic PacketCreator.java spawnPortal (line 1096):
//
//	p.writeInt(townId)      — 4 bytes LE uint32
//	p.writeInt(targetId)    — 4 bytes LE uint32
//	p.writePos(pos)         — writeShort(x), writeShort(y) [ByteBufOutPacket line 85-87]
//
// Total: 12 bytes. Unbranched across all versions (no structural delta known).
//
// For town-side door REMOVAL use RemoveTownDoor (remove_town.go), NOT this
// encoder. Cosmic removeDoor(town=true) omits writePos (8-byte body); this
// encoder always writes position (12 bytes) and must NOT be used for removal.
//
// IDA gms_v83: CWvsContext::OnTownPortal (0xa226a6) reads Decode4(townId) →
// Decode4(targetId), then Decode2(x) → Decode2(y) ONLY when neither id is
// MapId.NONE (999999999) — so a live portal is exactly the 12-byte layout below.
//
// packet-audit:verify packet=door/clientbound/SpawnPortal version=gms_v83 ida=0xa226a6
func TestSpawnPortal(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	townMapId := _map.Id(100000000)
	targetMapId := _map.Id(910000000)
	m := NewSpawnPortal(townMapId, targetMapId, -100, 300)

	// Golden wire layout (little-endian):
	//   writeInt(100000000)   → 0x00 0xE1 0xF5 0x05  (0x05F5E100 LE)
	//   writeInt(910000000)   → 0x80 0x7F 0x3D 0x36  (0x363D7F80 LE)
	//   writeShort(-100)      → 0x9C 0xFF
	//   writeShort(300)       → 0x2C 0x01
	want := []byte{
		0x00, 0xE1, 0xF5, 0x05, // townMapId = 100000000 LE
		0x80, 0x7F, 0x3D, 0x36, // targetMapId = 910000000 LE
		0x9C, 0xFF,             // x = -100 LE int16
		0x2C, 0x01,             // y = 300 LE int16
	}

	// v83 golden bytes
	v83ctx := pt.CreateContext("GMS", 83, 1)
	v83 := m.Encode(l, v83ctx)(nil)
	if !bytes.Equal(v83, want) {
		t.Errorf("SpawnPortal v83 golden bytes mismatch\n got: % x\nwant: % x", v83, want)
	}

	// Cross-version equality: all known versions must produce identical bytes.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := m.Encode(l, ctx)(nil)
			if !bytes.Equal(got, v83) {
				t.Errorf("SpawnPortal %s differs from v83\n got: % x\nv83: % x", v.Name, got, v83)
			}
		})
	}
}
