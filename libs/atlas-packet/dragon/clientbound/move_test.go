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

// v84: CDragon::OnMove (renamed from sub_507AC6 during this verification)
// @0x507ac6 (GMS_v84.1_U_DEVM.i64, session 5881cf84) is a single delegating
// call: CMovePath::OnMovePacket(this[55]!=0 ? this[55]-12+428 : 428, a2) —
// identical shape to v83/v95. Reached via CUser::OnDragonPacket (renamed
// from sub_9704B9) @0x9704b9's case 186 (0xBA) branch (gated on
// this[2074]).
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=gms_v84 ida=0x507ac6
func TestDragonMoveBytes_v84(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 84, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v84 move bytes = % X, want % X", got, want)
	}
}

// v87: CDragon::OnMove @0x520c71 (GMSv87_4GB.exe.i64, session d51ecbd3) is a
// single delegating call: CMovePath::OnMovePacket(*(this+55)!=0 ?
// *(this+55)-12+572 : 572, a2) — same shape as v83/v84/v95 (only the fallback
// constant differs per version's struct layout; the wire shape is
// unaffected). Reached via CUser::OnDragonPacket @0x9b3880's a2==0xC3 (195)
// branch (gated on this[2203]).
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=gms_v87 ida=0x520c71
func TestDragonMoveBytes_v87(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 87, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 move bytes = % X, want % X", got, want)
	}
}

// v92: CDragon::OnMove @0x505560 (GMS_v92_1_DEVM.exe.i64, session acdfccff)
// is a single delegating call: CMovePath::OnMovePacket(this[55]!=0 ?
// this[55]-12+580 : 580, a2, 0) — same shape as v83/v84/v87/v95 (only the
// fallback constant differs per version's struct layout; the wire shape is
// unaffected). Reached via CUser::OnDragonPacket (renamed from sub_8CE880
// during this verification) @0x8ce880's a2==210 branch (gated on
// this[2792]).
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=gms_v92 ida=0x505560
func TestDragonMoveBytes_v92(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 92, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v92 move bytes = % X, want % X", got, want)
	}
}

// JMS185: CDragon::OnMove @0x52f94f (MapleStory_dump_SCY.exe.i64 JMS185,
// session b6864e54) is a single delegating call:
// CMovePath::OnMovePacket(*(this+55)!=0 ? *(this+55)-12+580 : 580, iPacket,
// 0) — same shape as v83/v84/v87/v92/v95 (only the fallback constant differs
// per version's struct layout; the wire shape is unaffected). Reached via
// CUser::OnDragonPacket @0x9f822f's nType==0xBC (188) branch (gated on an
// existing CDragon), itself routed by CUserPool::OnUserCommonPacket
// @0xa440d7's range test `v5>=187 && v5<=189` @0xa44200.
//
// packet-audit:verify packet=dragon/clientbound/DragonMove version=jms_v185 ida=0x52f94f
func TestDragonMoveBytes_jms185(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("JMS", 185, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("jms_v185 move bytes = % X, want % X", got, want)
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
