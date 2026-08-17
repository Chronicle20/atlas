package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v87 (CWvsContext::SendSelectNpcItemUseRequest @0xaa5a85): COutPacket(0x72)
// @0xaa5b4b + Encode2(a2=source)@0xaa5b5d + Encode4(arg4=itemId)@0xaa5b68 —
// matches NpcItemUse.Encode field order exactly (source, itemId), no leading
// updateTime, no version gate. Not to be confused with the sibling
// SendScriptRunItemRequest @0xa9f3d2 (opcode 0x51, has updateTime).
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v87 ida=0xaa5a85
func TestNpcItemUseBytesV87(t *testing.T) {
	ctx := test.CreateContext("GMS", 87, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16 /*0xaa5b5d*/
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32 /*0xaa5b68*/
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 = % X, want % X", got, want)
	}
}

// gms_v79 (CWvsContext::SendSelectNpcItemUseRequest @0x95b96c,
// GMS_v79_1_DEVM.exe.i64): the guard `(a3/10000==545 || a3/10000==239) &&
// get_update_time(...)` selects the send, then two refusal arms (field flag
// bit 18; CUniqueModeless dialog open) return without sending anything.
// Otherwise, in read order —
//
//	COutPacket::COutPacket((COutPacket*)v13, 109)  /*0x95ba32*/ opcode 109 decimal = 0x6D
//	COutPacket::Encode2((COutPacket*)v13, a2)       /*0x95ba44*/ source (nPOS), int16 LE
//	COutPacket::Encode4((COutPacket*)v13, a3)       /*0x95ba4f*/ itemId (nItemID), uint32 LE
//
// No updateTime encode call appears anywhere in the decompile — matches the
// v61/v83/v84/v87 fixtures above.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v79 ida=0x95b96c
func TestNpcItemUseWireLayout_GMSv79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16      (Encode2 @0x95ba44)
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32     (Encode4 @0x95ba4f)
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// Byte round-trip over the invariant serverbound body. Identical on all nine
// versions that carry the opcode (v61 through jms_v185), so no version gating.
//
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// THERE IS NO updateTime. The sibling ScriptedItem codec in this package leads
// with one; copying its prologue here misaligns every subsequent read. This is
// the single most likely defect in this pair.
//
// Client gate on every version:
//
//	(nItemID / 10000 == 545 || nItemID / 10000 == 239) && CanSendExclRequest(200, 0)
//
// plus two refusal arms that emit a chat message and send nothing (field flag
// bit 18 set; a CUniqueModeless dialog already open).
//
// ABSENT from gms_v12 and gms_v48 — confirmed by instruction scan for
// `cmp ,545` / `cmp ,239`, not by a missing symbol.
func TestNpcItemUseRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NpcItemUse{source: 3, itemId: 2390001}
			output := NpcItemUse{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// Guards the no-updateTime invariant explicitly: the encoded frame must be
// exactly 6 bytes. A stray leading updateTime makes it 10.
//
// gms_v61 hand-computed against CWvsContext::SendSelectNpcItemUseRequest
// (0x83778d, GMS_v61.1_U_DEVM.exe.i64):
//
//	COutPacket::COutPacket(&v9, 102)      /*0x837851*/ -> opcode 0x066, no wire bytes of its own
//	COutPacket::Encode2(&v9, a2)          /*0x837863*/ -> source, int16 LE  -> bytes[0:2]
//	COutPacket::Encode4(&v9, a3)          /*0x83786e*/ -> itemId, int32 LE -> bytes[2:6]
//
// a2/a3 are the function's own params (source slot, itemId) with no
// intervening writes between the ctor and the two Encode calls, so the frame
// is exactly {source LE16, itemId LE32} = 6 bytes, matching `want` below.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v61 ida=0x83778d
//
// Decompiled CWvsContext::SendSelectNpcItemUseRequest (gms_v83, IDA 0xa10075):
//
//	COutPacket::COutPacket(&v10, 0x6F)            // 0xa1013b opcode 0x06F
//	COutPacket::Encode2(&v10, a2)                  // 0xa1014d source (nPOS), int16 LE
//	COutPacket::Encode4(&v10, a1)                  // 0xa10158 itemId (nItemID), uint32 LE
//
// No updateTime encode call appears anywhere in the decompile.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v83 ida=0xa10075
//
// gms_v84 wire-verified against CWvsContext::SendSelectNpcItemUseRequest
// (sub_A5A4B2 @ 0xa5a4b2, GMS_v84.1_U_DEVM.i64) — byte-identical layout:
//
//	COutPacket::COutPacket((COutPacket *)v10, 111);   /*0xa5a578*/ opcode 111 == 0x06F, matches registry
//	COutPacket::Encode2((COutPacket *)v10, a2);        /*0xa5a58a*/ source (nPOS), int16 LE
//	COutPacket::Encode4((COutPacket *)v10, a3);        /*0xa5a595*/ itemId (nItemID), uint32 LE
//
// No updateTime encode call appears anywhere in the decompile.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v84 ida=0xa5a4b2
// TestNpcItemUseWireLayout_JMSv185 hand-computes the wire bytes from the
// live jms_v185 decompile of CWvsContext::SendSelectNpcItemUseRequest
// (0xaf43ee, MapleStory_dump_SCY.exe.i64 — the clean session, not SMC-blocked):
//
//	COutPacket::COutPacket(v11, 0x6A)             // 0xaf44b4 opcode 0x06A, matches registry
//	COutPacket::Encode2(v11, nPOS)                 // 0xaf44c6 source (nPOS), int16 LE
//	COutPacket::Encode4(v11, nItemID)              // 0xaf44d1 itemId (nItemID), uint32 LE
//
// No updateTime encode call appears anywhere in the decompile. Two client-side
// refusal arms (field flag bit 18 set; CUniqueModeless dialog open) return
// before reaching COutPacket::COutPacket — they emit a chat StringPool message
// and send nothing, not additional wire fields.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=jms_v185 ida=0xaf43ee
func TestNpcItemUseWireLayout_JMSv185(t *testing.T) {
	ctx := test.CreateContext("JMS", 185, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16      (Encode2 @0xaf44c6)
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32     (Encode4 @0xaf44d1)
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestNpcItemUseWireLayoutHasNoUpdateTime(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestNpcItemUseWireLayout_GMSv72 hand-computes the wire bytes from the live
// gms_v72 decompile of CWvsContext::SendSelectNpcItemUseRequest at 0x90a5ac
// (named in this IDB):
//
//	if ((a3/10000==545 || a3/10000==239) && sub_4DBE16(200,0))   /*0x90a5e3*/  gate
//	  ... two early-return refusal arms (field flag bit 18; dword_A9F430 modeless-open) — no send
//	COutPacket::COutPacket((COutPacket*)v13, 110)                /*0x90a672*/  opcode 110 decimal = 0x6E
//	COutPacket::Encode2(v13, a2)                                 /*0x90a684*/  int16 source slot (a2 param)
//	COutPacket::Encode4(v13, a3)                                 /*0x90a68f*/  uint32 itemId (a3 param)
//
// No leading updateTime read anywhere in this function — confirms the sibling
// contrast documented above.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v72 ida=0x90a5ac
func TestNpcItemUseWireLayout_GMSv72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16   (Encode2 @0x90a684)
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32  (Encode4 @0x90a68f)
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestNpcItemUseWireLayout_GMSv92 hand-computes the wire bytes from the live
// gms_v92 decompile of CWvsContext::SendSelectNpcItemUseRequest at 0x9aff40
// (GMS_v92_1_DEVM.exe.i64, IDA session acdfccff):
//
//	if ((a3/10000==545 || a3/10000==239) && !this[2094])          /*0x9aff8f*/  gate
//	  ... two early-return refusal arms (field flag bit 18 set; dword_C2EF9C
//	      CUniqueModeless dialog open) — no send
//	COutPacket::COutPacket((COutPacket*)&v15, 0x7Au)               /*0x9b00c8*/  opcode 0x07A, matches registry
//	COutPacket::Encode2((COutPacket*)&v15, a2)                     /*0x9b00de*/  source (nPOS), int16 LE
//	COutPacket::Encode4(&v15, v4)                                  /*0x9b00e8*/  itemId (nItemID, v4=a3), uint32 LE
//
// No leading updateTime read anywhere in this function — matches the
// v61/v72/v79/v83/v84/v87/jms_v185 fixtures above.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v92 ida=0x9aff40
func TestNpcItemUseWireLayout_GMSv92(t *testing.T) {
	ctx := test.CreateContext("GMS", 92, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16   (Encode2 @0x9b00de)
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32  (Encode4 @0x9b00e8)
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestNpcItemUseWireLayout_GMSv95 hand-computes the wire bytes from the live
// gms_v95 decompile of CWvsContext::SendSelectNpcItemUseRequest at 0x9da430
// (GMS_v95.0_U_DEVM.exe.i64, IDA session 79906a1e):
//
//	if ((nItemID/10000==545 || nItemID/10000==239) && ...)     /*0x9da4ba*/  gate
//	  ... two early-return refusal arms (field flag bit 18 set on the current
//	      CField stage; CUniqueModeless dialog already open) — no send
//	COutPacket::COutPacket(&oPacket, 123)                       /*0x9da5b7*/  opcode 123 == 0x07B, matches registry
//	COutPacket::Encode2(&oPacket, nPOS)                         /*0x9da5cd*/  source (nPOS), int16 LE
//	COutPacket::Encode4(&oPacket, v4)                           /*0x9da5d7*/  itemId (v4 = nItemID), uint32 LE
//
// No leading updateTime read anywhere in this function — matches the
// v61/v72/v79/v83/v84/v87/v92/jms_v185 fixtures above.
//
// packet-audit:verify packet=inventory/serverbound/InventoryNpcItemUse version=gms_v95 ida=0x9da430
func TestNpcItemUseWireLayout_GMSv95(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := test.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16   (Encode2 @0x9da5cd)
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32  (Encode4 @0x9da5d7)
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}
