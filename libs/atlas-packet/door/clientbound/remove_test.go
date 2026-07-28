package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestRemoveDoor pins the full wire body of removeDoor's non-town path
// (REMOVE_DOOR clientbound, opcode chosen by config at runtime).
//
// the removeDoor, town=false branch:
//
//	p.writeByte(0) — 1 constant byte
//	p.writeInt(ownerId) — 4 bytes LE uint32
//
// Total: 5 bytes. Unbranched across all versions (no structural delta known).
//
// The town=true path in the v83 client emits SPAWN_PORTAL with two NONE map-ids and
// NO position (8-byte body); that is modelled by RemoveTownDoor (remove_town.go),
// NOT SpawnPortal — see remove_town_test.go for the 8-byte invariant test.
//
// IDA gms_v83: CTownPortalPool::OnTownPortalRemoved (0x7be064, dispatched from
// CTownPortalPool::OnPacket case 0x114) reads Decode1(animate flag) →
// Decode4(ownerId) and reads nothing further — the 5-byte layout pinned below.
//
// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v83 ida=0x7be064
// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v84 ida=0x7e40de
// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v87 ida=0x811487
// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v95 ida=0x761920
// packet-audit:verify packet=door/clientbound/RemoveDoor version=jms_v185 ida=0x84195b
func TestRemoveDoor(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	m := NewRemoveDoor(2500)

	// Golden wire layout (little-endian):
	// writeByte(0) → 0x00
	// writeInt(2500) → 0xC4 0x09 0x00 0x00
	want := []byte{
		0x00,                   // constant zero byte
		0xC4, 0x09, 0x00, 0x00, // ownerId = 2500 LE
	}

	// v83 golden bytes
	v83ctx := pt.CreateContext("GMS", 83, 1)
	v83 := m.Encode(l, v83ctx)(nil)
	if !bytes.Equal(v83, want) {
		t.Errorf("RemoveDoor v83 golden bytes mismatch\n got: % x\nwant: % x", v83, want)
	}

	// Cross-version equality: all known versions must produce identical bytes
	// (no structural branch implemented — single the v83 client layout applies to all).
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := m.Encode(l, ctx)(nil)
			if !bytes.Equal(got, v83) {
				t.Errorf("RemoveDoor %s differs from v83\n got: % x\nv83: % x", v.Name, got, v83)
			}
		})
	}
}
