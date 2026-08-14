package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v87 (CWvsContext::SendScriptRunItemRequest @0xa9f3d2): COutPacket(0x51)
// @0xa9f41f + Encode4(get_update_time())@0xa9f431 + Encode2(a2=source)@0xa9f43c
// + Encode4(arg4=itemId)@0xa9f447 — matches ScriptedItem.Encode field order
// exactly (updateTime, source, itemId), no version gate.
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v87 ida=0xa9f3d2
func TestScriptedItemBytesV87(t *testing.T) {
	ctx := test.CreateContext("GMS", 87, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32 /*0xa9f431*/
		0x06, 0x05, //             source, little-endian int16      /*0xa9f43c*/
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32     /*0xa9f447*/
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 = % X, want % X", got, want)
	}
}

// gms_v79 (GMS_v79_1_DEVM.exe.i64, sub_955840 @0x955840 — IDB mislabels the
// symbol, distrust-IDB-names caveat §10): the guard `a3/10000 == 243` selects
// the ScriptedItem branch, then in read order —
//
//	COutPacket::COutPacket((COutPacket*)v9, 76)   /*0x955879*/ opcode 76 decimal = 0x4C
//	v6 = get_update_time()                         /*0x95586b*/ -> updateTime
//	COutPacket::Encode4((COutPacket*)v9, v6)       /*0x95588b*/ updateTime, uint32 LE
//	COutPacket::Encode2((COutPacket*)v9, a2)       /*0x955896*/ source (nPOS), int16 LE
//	COutPacket::Encode4((COutPacket*)v9, a3)       /*0x9558a1*/ itemId (nItemID), uint32 LE
//
// Body layout matches the v83/v84/v72/v87 fixtures above (no version gate on
// the codec).
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v79 ida=0x955840
func TestScriptedItemWireLayout_GMSv79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32 (Encode4 @0x95588b)
		0x06, 0x05, //             source, little-endian int16      (Encode2 @0x955896)
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32     (Encode4 @0x9558a1)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// Byte round-trip over the invariant serverbound body. The client body is
// byte-identical on every version that carries the opcode — a full sweep of all
// ten IDBs found no divergence (task-230 design §1.1), so no version gating is
// required or permitted.
//
//	Encode4(get_update_time())   // uint32 update time
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// Gated client-side on nItemID / 10000 == 243 under CanSendExclRequest(500, 0).
// v83+ additionally guards on CWvsContext::IsAbleToConsume, which v72/v79 lack;
// that is a client-side convenience check the server must not rely on.
// v95 alone also whitelists nItemID == 3994225 (an Install/Setup item) — out of
// scope per design D-3 and rejected server-side.
//
// The op is ABSENT from gms_v12, gms_v48, and gms_v61 (design §1.1 absence
// evidence: dense Send*ItemUseRequest export sets with no SendScriptRunItemRequest).
//
// Per-cell verify markers (the packet-audit "verify" annotation) are added in
// the verification task; this round-trip alone is NOT a verification.
func TestScriptedItemRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ScriptedItem{updateTime: 0x1A2B3C4D, source: 7, itemId: 2430008}
			output := ScriptedItem{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// The field ORDER is the defect this guards. The sibling NpcItemUse codec has
// no leading updateTime; reading these two files side by side and copying the
// wrong prologue misaligns every subsequent field. Assert the exact bytes.
//
// Decompiled CWvsContext::SendScriptRunItemRequest (gms_v83, IDA 0xa09b26):
//
//	COutPacket::COutPacket(&v6, 0x4E)             // 0xa09b73 opcode 0x04E
//	update_time = get_update_time()               // 0xa09b7c
//	COutPacket::Encode4(&v6, update_time)          // 0xa09b85 updateTime, uint32 LE
//	COutPacket::Encode2(&v6, a2)                   // 0xa09b90 source (nPOS), int16 LE
//	COutPacket::Encode4(&v6, a3)                   // 0xa09b9b itemId (nItemID), uint32 LE
//
// gms_v84 wire-verified against CWvsContext::SendScriptRunItemRequest
// (sub_A53F08 @ 0xa53f08, GMS_v84.1_U_DEVM.i64) — byte-identical layout:
//
//	COutPacket::COutPacket((COutPacket *)v11, 78);            /*0xa53f55*/ opcode 78 == 0x04E, matches registry
//	v7 = sub_9C7771(v6, v5);                                   /*0xa53f5e*/ get_update_time() -> uint32
//	COutPacket::Encode4((COutPacket *)v11, v7);                /*0xa53f67*/ updateTime, uint32 LE
//	COutPacket::Encode2((COutPacket *)v11, a2);                /*0xa53f72*/ source (nPOS), int16 LE
//	COutPacket::Encode4((COutPacket *)v11, a3);                 /*0xa53f7d*/ itemId (nItemID), uint32 LE
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v83 ida=0xa09b26
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v84 ida=0xa53f08
func TestScriptedItemWireLayout(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32
		0x06, 0x05, //             source, little-endian int16
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestScriptedItemWireLayout_JMSv185 hand-computes the wire bytes from the
// live jms_v185 decompile of CWvsContext::SendScriptRunItemRequest
// (0xaee7ce, MapleStory_dump_SCY.exe.i64 — the clean session, not SMC-blocked):
//
//	COutPacket::COutPacket(v7, 70)                // 0xaee81b opcode 70 decimal = 0x046, matches registry
//	update_time = get_update_time()                // 0xaee824
//	COutPacket::Encode4(v7, update_time)           // 0xaee82d updateTime, uint32 LE
//	COutPacket::Encode2(v7, nPOS)                  // 0xaee838 source (nPOS), int16 LE
//	COutPacket::Encode4(v7, nItemID)               // 0xaee843 itemId (nItemID), uint32 LE
//
// Body layout matches the v83/v84/v72 fixtures above — jms_v185 carries no
// divergent gate ahead of Encode4/Encode2/Encode4 (the itemId/10000==243 and
// IsAbleToConsume gates are client-side refusal arms, not wire fields).
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=jms_v185 ida=0xaee7ce
func TestScriptedItemWireLayout_JMSv185(t *testing.T) {
	ctx := test.CreateContext("JMS", 185, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32 (Encode4 @0xaee82d)
		0x06, 0x05, //             source, little-endian int16      (Encode2 @0xaee838)
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32     (Encode4 @0xaee843)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestScriptedItemWireLayout_GMSv72 hand-computes the wire bytes from the
// live gms_v72 decompile of the sender at 0x9044d8 (unnamed in the IDB — the
// registry fname CWvsContext::SendScriptRunItemRequest is confirmed by the
// itemId/10000==243 gate at this same address, per §10 "distrust IDB names"):
//
//	COutPacket::COutPacket((COutPacket*)v9, 77)          /*0x904511*/  opcode 77 decimal = 0x4D
//	get_update_time(v5) -> COutPacket::Encode4(v9, v6)    /*0x90451a-0x904523*/  uint32 updateTime
//	COutPacket::Encode2(v9, a2)                           /*0x90452e*/  int16 source slot (a2 param)
//	COutPacket::Encode4(v9, a3)                           /*0x904539*/  uint32 itemId (a3 param)
//
// Body layout matches the v83 fixture above — v72 carries no divergent gate
// ahead of Encode4/Encode2/Encode4.
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v72 ida=0x9044d8
func TestScriptedItemWireLayout_GMSv72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32 (Encode4 @0x904523)
		0x06, 0x05, //             source, little-endian int16      (Encode2 @0x90452e)
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32     (Encode4 @0x904539)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestScriptedItemWireLayout_GMSv92 hand-computes the wire bytes from the
// live gms_v92 decompile of the sender at 0x9b3da0 (GMS_v92_1_DEVM.exe.i64,
// IDA session acdfccff), confirmed by the itemId/10000==243 gate at the top
// of the function (registry fname CWvsContext::SendScriptRunItemRequest):
//
//	COutPacket::COutPacket((COutPacket*)&v9, 0x55u)  /*0x9b3e39*/  opcode 0x055, matches registry
//	v7 = sub_936E80(v6)                               /*0x9b3e46*/  get_update_time() -> uint32
//	COutPacket::Encode4(&v9, v7)                      /*0x9b3e50*/  updateTime, uint32 LE
//	COutPacket::Encode2((COutPacket*)&v9, a2)         /*0x9b3e5e*/  source (nPOS), int16 LE
//	COutPacket::Encode4(&v9, a3)                      /*0x9b3e68*/  itemId (nItemID), uint32 LE
//
// Body layout matches the v72/v83/v84/v87 fixtures above — v92 carries no
// divergent gate ahead of Encode4/Encode2/Encode4.
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v92 ida=0x9b3da0
func TestScriptedItemWireLayout_GMSv92(t *testing.T) {
	ctx := test.CreateContext("GMS", 92, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32 (Encode4 @0x9b3e50)
		0x06, 0x05, //             source, little-endian int16      (Encode2 @0x9b3e5e)
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32     (Encode4 @0x9b3e68)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestScriptedItemWireLayout_GMSv95 hand-computes the wire bytes from the
// live gms_v95 decompile of CWvsContext::SendScriptRunItemRequest (0x9de7a0,
// GMS_v95.0_U_DEVM.exe.i64, IDA session 79906a1e), confirmed by the guard
// `nItemID / 10000 == 243 || nItemID == 3994225` at the top of the function
// (registry fname CWvsContext::SendScriptRunItemRequest; the extra ==3994225
// arm is the v95-only Install/Setup-item whitelist, a client-side gate that
// does not add or reorder wire fields):
//
//	COutPacket::COutPacket(&oPacket, 84)          /*0x9de840*/  opcode 84 == 0x054, matches registry
//	update_time = get_update_time()               /*0x9de84d*/  uint32 update time
//	COutPacket::Encode4(&oPacket, update_time)    /*0x9de857*/  updateTime, uint32 LE
//	COutPacket::Encode2(&oPacket, nPOS)           /*0x9de865*/  source (nPOS), int16 LE
//	COutPacket::Encode4(&oPacket, nItemID)        /*0x9de86f*/  itemId (nItemID), uint32 LE
//
// Body layout matches the v72/v79/v83/v84/v87/v92/jms_v185 fixtures above —
// v95 carries no divergent gate ahead of Encode4/Encode2/Encode4.
//
// packet-audit:verify packet=inventory/serverbound/InventoryScriptedItem version=gms_v95 ida=0x9de7a0
func TestScriptedItemWireLayout_GMSv95(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32 (Encode4 @0x9de857)
		0x06, 0x05, //             source, little-endian int16      (Encode2 @0x9de865)
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32     (Encode4 @0x9de86f)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}
