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

// v84: CUser::OnDragonPacket (renamed from sub_9704B9 during this
// verification) @0x9704b9 (GMS_v84.1_U_DEVM.i64, session 5881cf84) handles
// only a2==185 (spawn, CDragon::OnCreated) and this[2074]&&a2==186 (move,
// CDragon::OnMove) — there is no case for 187 (REMOVE_DRAGON). The outer
// dispatcher CUserPool::OnUserCommonPacket @0x9b2443 routes the range
// v6>=185 && v6<=187 into this function (@0x9b2443/0x9b244b), so 187 enters
// and falls through both branches unhandled — same routed-but-unhandled
// shape as v83 and v95. See docs/packets/registry/gms_v84.yaml:1254 (already
// recorded prior to this verification) for the full decompile citation.
//
// packet-audit:verify packet=dragon/clientbound/DragonRemove version=gms_v84 ida=0x9704b9
func TestDragonRemoveBytes_v84(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 84, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("v84 remove bytes = % X, want 92 10 00 00", got)
	}
}

// v87: CUser::OnDragonPacket @0x9b3880 (GMSv87_4GB.exe.i64, session d51ecbd3)
// handles only a2==0xC2 (194, spawn -> CDragon::OnCreated @0x5200ed) and
// this[2203]&&a2==0xC3 (195, move -> CDragon::OnMove @0x520c71) — THERE IS NO
// CASE FOR 0xC4 (196, REMOVE_DRAGON). The outer dispatcher
// CUserPool::OnUserCommonPacket @0x9f7387 routes the range
// `v6>=0xC2 && v6<=0xC4` @0x9f7430 into OnDragonPacket (call @0x9f7438), so
// 196 enters the function and falls through both branches unhandled — no
// field is read, no dragon state changes. Same routed-but-unhandled shape as
// v83/v84/v95 (docs/packets/registry/gms_v84.yaml:1254). Sending
// REMOVE_DRAGON is correct and harmless; the client silently discards it.
//
// packet-audit:verify packet=dragon/clientbound/DragonRemove version=gms_v87 ida=0x9b3880
func TestDragonRemoveBytes_v87(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 87, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("v87 remove bytes = % X, want 92 10 00 00", got)
	}
}

// v92: CUser::OnDragonPacket (renamed from sub_8CE880 during this
// verification) @0x8ce880 (GMS_v92_1_DEVM.exe.i64, session acdfccff)
// switches on a2: a2==209 -> spawn (CDragon::OnCreated @0x5084c0),
// this[2792]&&a2==210 -> move (CDragon::OnMove @0x505560) — THERE IS NO
// CASE FOR 211 (REMOVE_DRAGON). The outer dispatcher
// CUserPool::OnUserCommonPacket @0x929750 routes the range
// `(unsigned int)(v6 - 209) <= 2` @0x9298b9 into OnDragonPacket
// (call @0x9298bf), so 211 enters the function and falls through both
// branches unhandled — no field is read, no dragon state changes. Same
// routed-but-unhandled shape as v83/v84/v87/v95
// (docs/packets/registry/gms_v84.yaml:1254). Sending REMOVE_DRAGON is
// correct and harmless; the client silently discards it.
//
// packet-audit:verify packet=dragon/clientbound/DragonRemove version=gms_v92 ida=0x8ce880
func TestDragonRemoveBytes_v92(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 92, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("v92 remove bytes = % X, want 92 10 00 00", got)
	}
}

func TestDragonRemoveRoundTrip(t *testing.T) {
	var out DragonRemove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
