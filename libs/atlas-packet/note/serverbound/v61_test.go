package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestOperationDiscardV61Body pins the gms_v61 NOTE_ACTION discard wire
// (serverbound op 0x77 / 119).
//
// Re-decompiled during this verification pass (session 965202bf,
// GMS_v61.1_U_DEVM.exe.i64) to confirm the corrected codec shape — a prior
// revision of this fixture dropped the specialCount header byte, which
// crashed live clients (the decode desynced on the following field). The
// FULL body, cited fresh from CMemoListDlg::SetRet @0x5ad50c, delete-on-close
// path (a2==1||a2==2, user confirms YesNo):
//
//	COutPacket::COutPacket(v25, 119)     @0x5ad58d → opcode 0x77 (registry op 119).
//	Encode1(1u)                          @0x5ad59a → mode byte = 1 (discard);
//	                                                 consumed by the
//	                                                 NoteOperationHandle
//	                                                 dispatcher, NOT part of
//	                                                 this codec's body.
//	Encode1(totalCount)                  @0x5ad5b4 → total local memo-list length.
//	[loop @0x5ad5bb-0x5ad5d6 counts entries whose flag == 2]
//	Encode1(specialCount)                @0x5ad5e2 → # memos flagged 2 (gift/
//	                                                 reward), counted BEFORE
//	                                                 the free-slot budget is
//	                                                 applied. This is a
//	                                                 header byte on EVERY
//	                                                 version, not jms-only.
//	Encode1(emptySlotCount)              @0x5ad5ed → inbox ETC free-slot budget.
//	per memo (loop @0x5ad5f5-0x5ad6e5):
//	  flag == 2 (special):
//	    budget <= 0                      @0x5ad641 → entry SKIPPED, zero bytes
//	                                                 written (client shows a
//	                                                 "slot full" notice instead).
//	    else: Encode4(id) @0x5ad6a9 + Encode1(flag) @0x5ad6be
//	          + Encode4(parsedSN)        @0x5ad6c9  ← trailing reward/mesos
//	                                                 value (1 extra int32 on
//	                                                 GMS); budget--.
//	  else (normal): Encode4(id) @0x5ad61f + Encode1(flag) @0x5ad634.
//
// This is byte-for-byte the same field order as the VERIFIED gms_v72 sender
// CMemoListDlg::SetRet @0x5fb443; the only differences are the opcode (119 vs
// 129, Δ-10, already in the registry) and the special-flag sentinel (v61
// flag==2 vs v72 flag==3, per noteDiscardSpecialFlag) — neither is a wire
// layout change.
//
// Fixture: count=2, specialCount=1, emptySlotCount=3, entries
// [{100,flag:1(normal)},{200,flag:2(special),extra1:500}]. WriteInt =
// uint32-LE; WriteByte = one byte.
//
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v61 ida=0x5ad50c
func TestOperationDiscardV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := OperationDiscard{
		count:          2,
		specialCount:   1,
		emptySlotCount: 3,
		entries: []DiscardEntry{
			{id: 100, flag: 1},
			{id: 200, flag: 2, marriageNumber: 500},
		},
	}
	want := []byte{
		0x02,                   // count = 2 (totalCount Encode1 @0x5ad5b4)
		0x01,                   // specialCount = 1 (Encode1 @0x5ad5e2)
		0x03,                   // emptySlotCount = 3 (Encode1 @0x5ad5ed)
		0x64, 0x00, 0x00, 0x00, // entry[0].id = 100 (Encode4 @0x5ad61f)
		0x01,                   // entry[0].flag = 1 (Encode1 @0x5ad634, normal)
		0xC8, 0x00, 0x00, 0x00, // entry[1].id = 200 (Encode4 @0x5ad6a9)
		0x02,                   // entry[1].flag = 2 (Encode1 @0x5ad6be, special)
		0xF4, 0x01, 0x00, 0x00, // entry[1].extra1 = 500 (Encode4 @0x5ad6c9)
	}
	// v61's special flag (2) is resolved from the tenant version in code
	// (discardSpecialFlag: GMS<=61 -> 2), not tenant config.
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 NoteOperationDiscard golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
