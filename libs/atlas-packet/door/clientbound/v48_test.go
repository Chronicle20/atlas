package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 town-portal byte fixtures. Opcodes come from CTownPortalPool::OnPacket
// @0x5e318d, which routes 206 -> OnTownPortalCreated and 207 -> OnTownPortalRemoved
// (v61 uses 212/213). Both are shape-stable against v83; no codec change needed.

// TestSpawnDoorBytesV48 — CTownPortalPool::OnTownPortalCreated @0x5e31b2 reads
// Decode1(launched) @0x5e31f2 and Decode4(ownerId) @0x5e31fa, then branches on
// whether the portal already exists in the pool (sub_5E41DC). BOTH arms read
// exactly two more shorts — the existing-portal arm at 0x5e35ae/0x5e35c7 and the
// new-portal arm at 0x5e36a6/0x5e36b6 — so four fields execute on any single
// path.
//
// This is worth spelling out: the checked-in export lists SIX ops for this
// function on v48/v61/v72/v79 because its linear walk visits both arms, which
// reads like the legacy clients want 4 bytes Atlas never sends. They do not. The
// codec is correct as written.
//
// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v48 ida=0x5e31b2
func TestSpawnDoorBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewSpawnDoor(0x01020304, 300, -400, true).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x01,                   // launched — Decode1 @0x5e31f2
		0x04, 0x03, 0x02, 0x01, // ownerId  — Decode4 @0x5e31fa
		0x2C, 0x01, // x = 300  — Decode2
		0x70, 0xFE, // y = -400 — Decode2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 spawn door:\n got % x\nwant % x", got, want)
	}
}

// TestRemoveDoorBytesV48 — CTownPortalPool::OnTownPortalRemoved @0x5e3b40 reads
// Decode1 then Decode4(ownerId).
//
// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v48 ida=0x5e3b40
func TestRemoveDoorBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewRemoveDoor(0x01020304).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x00, 0x04, 0x03, 0x02, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 remove door:\n got % x\nwant % x", got, want)
	}
}
