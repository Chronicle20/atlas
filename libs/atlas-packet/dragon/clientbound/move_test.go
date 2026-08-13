package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOVE_DRAGON is int ownerCharacterId + the raw CMovePath blob. The blob already
// begins with the start position, so it must NOT be written separately.
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=gms_v95 ida=0x50ad30
func TestDragonMoveIsOwnerIdPlusOpaqueBlob(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 95, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("move bytes = % X, want % X", got, want)
	}
}

// v83: CDragon::OnMove equivalent (renamed from sub_4FEF90 during this
// verification) @0x4fef90 (MapleStory_dump.exe.i64 v83, session 41f13e0d) is
// a single delegating call: CMovePath::OnMovePacket((this[55]!=0 ?
// this[55]-12+428 : 428), iPacket, v2) @0x4fef9e — same shape as v95's
// CDragon::OnMove. ownerCharacterId is consumed upstream by
// CUserPool::OnUserCommonPacket @0x972401 before the range dispatch
// `v6>=181 && v6<=183` @0x97249d routes into CUser::OnDragonPacket @0x93908f,
// whose 0xB6 (182) branch calls this function.
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=gms_v83 ida=0x4fef90
func TestDragonMoveBytes_v83(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 83, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 move bytes = % X, want % X", got, want)
	}
}

func TestDragonMoveRoundTrip(t *testing.T) {
	var out DragonMove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1),
		NewDragonMove(4242, []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 || len(out.RawMovement()) != 6 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
