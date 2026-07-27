package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestOperationDiscardV48Body pins the gms_v48 NOTE_ACTION discard wire
// (serverbound op 101 / 0x65).
//
// Re-decompiled during this verification pass (session 0bb5f11a,
// GMS_v48_1_DEVM.exe.i64) to confirm the corrected codec shape — a prior
// revision of this fixture dropped the specialCount header byte, which
// crashed live clients (the decode desynced on the following field). The
// FULL body, cited fresh from CMemoListDlg::SetRet @0x534dc4, delete-on-close
// path (a2==1||a2==2, user confirms YesNo @0x534e02):
//
//	COutPacket::COutPacket(v27, 101)     @0x534e45 → opcode 0x65 (registry op 101).
//	Encode1(1u)                          @0x534e52 → mode byte = 1 (discard);
//	                                                 consumed by the
//	                                                 NoteOperationHandle
//	                                                 dispatcher, NOT part of
//	                                                 this codec's body.
//	Encode1(totalCount)                  @0x534e6c → total local memo-list length.
//	[loop @0x534e73-0x534e8e counts entries whose flag == 2]
//	Encode1(specialCount)                @0x534e9a → # memos flagged 2 (gift/
//	                                                 reward), counted BEFORE
//	                                                 the free-slot budget is
//	                                                 applied. This is a
//	                                                 header byte on EVERY
//	                                                 version, not jms-only.
//	Encode1(emptySlotCount)              @0x534ea5 → inbox ETC free-slot budget.
//	per memo (loop @0x534ead-0x534f9c):
//	  flag == 2 (special):
//	    budget <= 0                      @0x534ef9 → entry SKIPPED, zero bytes
//	                                                 written (client shows a
//	                                                 "slot full" notice instead).
//	    else: Encode4(id) @0x534f60 + Encode1(flag) @0x534f75
//	          + Encode4(parsedSN)        @0x534f80  ← trailing reward/mesos
//	                                                 value (1 extra int32 on
//	                                                 GMS); budget--.
//	  else (normal): Encode4(id) @0x534ed7 + Encode1(flag) @0x534eec.
//
// This is byte-for-byte the same field order as the VERIFIED gms_v61 sender
// CMemoListDlg::SetRet @0x5ad50c (TestOperationDiscardV61Body); only the
// opcode (101 vs 119, Δ-18) differs — not a wire-layout change. Both v48/v61
// compare the special/delete flag against 2 (noteDiscardSpecialFlag); v72+
// compares against 3.
//
// Fixture: count=2, specialCount=1, emptySlotCount=3, entries
// [{100,flag:1(normal)},{200,flag:2(special),extra1:500}]. WriteInt =
// uint32-LE; WriteByte = one byte.
//
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v48 ida=0x534dc4
func TestOperationDiscardV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := OperationDiscard{
		count:          2,
		specialCount:   1,
		emptySlotCount: 3,
		entries: []DiscardEntry{
			{id: 100, flag: 1},
			{id: 200, flag: 2, claimValues: []uint32{500}},
		},
	}
	want := []byte{
		0x02,                   // count = 2 (totalCount Encode1 @0x534e6c)
		0x01,                   // specialCount = 1 (Encode1 @0x534e9a)
		0x03,                   // emptySlotCount = 3 (Encode1 @0x534ea5)
		0x64, 0x00, 0x00, 0x00, // entry[0].id = 100 (Encode4 @0x534ed7)
		0x01,                   // entry[0].flag = 1 (Encode1 @0x534eec, normal)
		0xC8, 0x00, 0x00, 0x00, // entry[1].id = 200 (Encode4 @0x534f60)
		0x02,                   // entry[1].flag = 2 (Encode1 @0x534f75, special)
		0xF4, 0x01, 0x00, 0x00, // entry[1].extra1 = 500 (Encode4 @0x534f80)
	}
	// v48's special flag (2) and claim-value count (1) come from the tenant
	// template's NoteOperationHandle.options.discard config (see
	// resolveDiscardShape), not a code literal.
	options := map[string]interface{}{
		DiscardConfigKey: map[string]interface{}{
			DiscardSpecialFlagKey:     float64(2),
			DiscardClaimValueCountKey: float64(1),
		},
	}
	if got := pt.Encode(t, ctx, input.Encode, options); !bytes.Equal(got, want) {
		t.Errorf("v48 NoteOperationDiscard golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
