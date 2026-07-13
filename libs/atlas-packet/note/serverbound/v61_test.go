package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestOperationDiscardV61Body pins the gms_v61 NOTE_ACTION discard wire
// (serverbound op 0x77 / 119).
//
// IDA-verified sender — CMemoListDlg::SetRet @0x5ad50c (GMS_v61.1_U_DEVM.exe,
// port 13338), delete-on-close path (a2==1||a2==2, user confirms YesNo):
//
//	COutPacket::COutPacket(v25, 119)  @0x5ad58d → opcode 0x77 (matches registry
//	                                              op 119 and template handler 0x77).
//	Encode1(1u)                       @0x5ad59a → action byte = 1 (discard).
//	Encode1(totalCount)               @0x5ad5b4 → total memo-list length.
//	Encode1(deleteCount)              @0x5ad5e2 → # memos flagged 2 (delete).
//	Encode1(emptySlotCount)           @0x5ad5ed → inbox empty-slot count.
//	per memo:  Encode4(id) @0x5ad61f + Encode1(flag) @0x5ad634          (keep)
//	           flag==2 → Encode4(id)+Encode1(flag)+Encode4(parsedSN)    (delete)
//
// This is byte-for-byte the same field order as the VERIFIED gms_v72 sender
// CMemoListDlg::SetRet @0x5fb443 (TestOperationDiscardRoundTrip); the only
// differences are the opcode (119 vs 129, Δ-10, already in the registry) and
// the delete-flag sentinel (v61 flag==2 vs v72 flag==3) — neither is a wire
// layout change.
//
// Atlas note/serverbound/OperationDiscard models the SERVER-side subset the
// verified v72 cell also models: count + emptySlotCount + a flat list of
// {id(4), flag(1)} entries (the leading action byte and totalCount are consumed
// by the NoteOperationHandle dispatcher before this codec runs; the conditional
// trailing parsedSN of delete-marked memos is not modeled — the same simplified
// model graded ✅ for v72). The codec is version-agnostic, so the v61 wire is
// identical to v72. WriteInt = uint32-LE; WriteByte = one byte. Fixture:
// count=2, emptySlotCount=3, entries [{100,1},{200,2}].
//
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v61 ida=0x5ad50c
func TestOperationDiscardV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := OperationDiscard{
		count:          2,
		emptySlotCount: 3,
		entries: []DiscardEntry{
			{id: 100, flag: 1},
			{id: 200, flag: 2},
		},
	}
	want := []byte{
		0x02,                   // count = 2 (deleteCount Encode1 @0x5ad5e2)
		0x03,                   // emptySlotCount = 3 (Encode1 @0x5ad5ed)
		0x64, 0x00, 0x00, 0x00, // entry[0].id = 100 (Encode4 @0x5ad61f)
		0x01,                   // entry[0].flag = 1 (Encode1 @0x5ad634)
		0xC8, 0x00, 0x00, 0x00, // entry[1].id = 200 (Encode4 @0x5ad61f)
		0x02, // entry[1].flag = 2 (Encode1 @0x5ad634)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 NoteOperationDiscard golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
