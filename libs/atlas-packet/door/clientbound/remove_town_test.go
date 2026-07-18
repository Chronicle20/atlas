package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestRemoveTownDoor pins the full wire body of the town-side door removal packet
// (SPAWN_PORTAL opcode, writer name RemoveTownDoor).
//
// the removeDoor, town=true branch:
//
//	p = OutPacket.create(SendOpcode.SPAWN_PORTAL)
//	p.writeInt(MapId.NONE) — 4 bytes LE uint32, MapId.NONE = 999999999
//	p.writeInt(MapId.NONE) — 4 bytes LE uint32
//
// Critically, NO writePos call follows — total body is exactly 8 bytes.
// SpawnPortal ALWAYS writes writePos (x,y) for 12 bytes total; using it for
// removal would corrupt the client read cursor with 4 spurious trailing bytes.
//
// Unbranched across all versions (no structural delta known).
//
// IDA gms_v83: CWvsContext::OnTownPortal (0xa226a6) guards the x/y reads with
// `if (townId != 999999999 && targetId != 999999999)`, so two NONE ids skip
// Decode2(x)/Decode2(y) entirely — confirming the 8-byte removal body below.
//
// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v83 ida=0xa226a6
// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v84 ida=0xa6dbb8
// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v87 ida=0xab9ef6
// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v95 ida=0x9f1330
// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=jms_v185 ida=0xb0977c
func TestRemoveTownDoor(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	m := NewRemoveTownDoor()

	// Golden wire layout (little-endian):
	// writeInt(999999999) → 0xFF 0xC9 0x9A 0x3B (_map.EmptyMapId, MapId.NONE)
	// writeInt(999999999) → 0xFF 0xC9 0x9A 0x3B
	// Total: 8 bytes. NO position bytes follow.
	want := []byte{
		0xFF, 0xC9, 0x9A, 0x3B, // EmptyMapId = 999999999 LE (first int)
		0xFF, 0xC9, 0x9A, 0x3B, // EmptyMapId = 999999999 LE (second int)
	}

	// v83 golden bytes
	v83ctx := pt.CreateContext("GMS", 83, 1)
	v83 := m.Encode(l, v83ctx)(nil)

	// Guard the "no position" invariant explicitly.
	if len(v83) != 8 {
		t.Fatalf("RemoveTownDoor body must be exactly 8 bytes (no position); got %d bytes: % x", len(v83), v83)
	}

	if !bytes.Equal(v83, want) {
		t.Errorf("RemoveTownDoor v83 golden bytes mismatch\n got: % x\nwant: % x", v83, want)
	}

	// Cross-version equality: all known versions must produce identical bytes
	// (no structural branch implemented — single the v83 client layout applies to all).
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := m.Encode(l, ctx)(nil)
			if len(got) != 8 {
				t.Fatalf("RemoveTownDoor %s body must be exactly 8 bytes; got %d bytes: % x", v.Name, len(got), got)
			}
			if !bytes.Equal(got, v83) {
				t.Errorf("RemoveTownDoor %s differs from v83\n got: % x\nv83: % x", v.Name, got, v83)
			}
		})
	}
}
