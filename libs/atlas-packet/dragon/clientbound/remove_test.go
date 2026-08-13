package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// REMOVE_DRAGON has no body: the only field is the owner character id, and the
// client has no handler arm for the opcode at all. CUser::OnDragonPacket
// (GMS v95.0 @0x8e5c00) switches on nType: 206 -> spawn, 207 -> move, and
// nothing else; the outer dispatcher CUserPool::OnUserCommonPacket routes
// 206..208 into it, so 208 (REMOVE_DRAGON) enters the function and falls
// through both branches unhandled. See docs/packets/registry/gms_v95.yaml's
// REMOVE_DRAGON note for the full decompile citation.
//
// packet-audit:verify packet=dragon/clientbound/DragonRemove version=gms_v95 ida=0x8e5c00
func TestDragonRemoveIsOwnerIdOnly(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("remove bytes = % X, want 92 10 00 00", got)
	}
}

// v83: CUser::OnDragonPacket @0x93908f (MapleStory_dump.exe.i64 v83, session
// 41f13e0d) switches on nType (a2): a2==0xB5 (181) -> spawn, else-if
// this[1996] && a2==0xB6 (182) -> move. THERE IS NO CASE FOR 0xB7 (183,
// REMOVE_DRAGON) — the outer dispatcher CUserPool::OnUserCommonPacket
// @0x972401 routes opcodes 181..183 into OnDragonPacket via the range test
// `v6>=181 && v6<=183` @0x97249d, so 183 enters this function and falls
// through both branches unhandled — same routed-but-unhandled shape as v84
// (docs/packets/registry/gms_v84.yaml:1254) and v95. See
// docs/packets/registry/gms_v83.yaml's REMOVE_DRAGON note for the full
// decompile citation.
//
// packet-audit:verify packet=dragon/clientbound/DragonRemove version=gms_v83 ida=0x93908f
func TestDragonRemoveBytes_v83(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 83, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("v83 remove bytes = % X, want 92 10 00 00", got)
	}
}

func TestDragonRemoveRoundTrip(t *testing.T) {
	var out DragonRemove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
