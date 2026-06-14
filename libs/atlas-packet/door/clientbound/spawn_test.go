package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestSpawnDoor pins the full wire body of spawnDoor (SPAWN_DOOR clientbound).
//
// Cosmic PacketCreator.java spawnDoor (line 1112):
//
//	p.writeBool(launched)       — 1 byte
//	p.writeInt(ownerid)         — 4 bytes LE uint32
//	p.writePos(pos)             — writeShort(x), writeShort(y) — 4 bytes
//
// Total: 9 bytes. Unbranched across all versions (no structural delta known).
//
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v83 ida=TODO
func TestSpawnDoor(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	// Input values: ownerId=1000, x=100, y=200, launched=true
	m := NewSpawnDoor(1000, 100, 200, true)

	// Golden wire layout (little-endian):
	//   writeBool(true)    → 0x01
	//   writeInt(1000)     → 0xE8 0x03 0x00 0x00
	//   writeShort(100)    → 0x64 0x00
	//   writeShort(200)    → 0xC8 0x00
	want := []byte{
		0x01,                   // launched = true
		0xE8, 0x03, 0x00, 0x00, // ownerid = 1000 LE
		0x64, 0x00,             // x = 100 LE short
		0xC8, 0x00,             // y = 200 LE short
	}

	// v83 golden bytes
	v83ctx := pt.CreateContext("GMS", 83, 1)
	v83 := m.Encode(l, v83ctx)(nil)
	if !bytes.Equal(v83, want) {
		t.Errorf("SpawnDoor v83 golden bytes mismatch\n got: % x\nwant: % x", v83, want)
	}

	// Cross-version equality: all known versions must produce identical bytes
	// (no structural branch implemented — single Cosmic layout applies to all).
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
