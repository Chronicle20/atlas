package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestOperationDiscardV48Body pins the gms_v48 NOTE_ACTION discard wire
// (serverbound op 101 / 0x65).
//
// IDA-verified sender — CMemoListDlg::SetRet @0x534dc4 (GMS_v48_1_DEVM.exe,
// port 13337), delete-on-close path (a2==1||a2==2, user confirms YesNo @0x534e02):
//
//	COutPacket::COutPacket(v27, 101) @0x534e45 → opcode 0x65 (matches registry
//	                                             op 101).
//	Encode1(1u)              @0x534e52 → action byte = 1 (discard).
//	Encode1(totalCount)      @0x534e6c → total memo-list length.
//	Encode1(deleteCount)     @0x534e9a → # memos flagged 2 (delete).
//	Encode1(emptySlotCount)  @0x534ea5 → inbox empty-slot count.
//	per memo: type!=2 → Encode4(id) @0x534ed7 + Encode1(flag) @0x534eec   (keep)
//	          type==2 → Encode4(id)+Encode1(flag)+Encode4(parsedSN)        (delete)
//
// This is byte-for-byte the same field order as the VERIFIED gms_v61 sender
// CMemoListDlg::SetRet @0x5ad50c (TestOperationDiscardV61Body); only the opcode
// (101 vs 119, Δ-18) differs — not a wire-layout change.
//
// Atlas note/serverbound/OperationDiscard models the SERVER-side subset the
// verified v61 cell also models: count(=deleteCount) + emptySlotCount + a flat
// list of {id(4), flag(1)} entries (the leading action byte and totalCount are
// consumed by the NoteOperationHandle dispatcher before this codec runs; the
// conditional trailing parsedSN of delete-marked memos is not modeled — the same
// simplified model graded ✅ for v61). The codec is version-agnostic, so the v48
// wire is identical to v61. WriteInt = uint32-LE; WriteByte = one byte. Fixture:
// count=2, emptySlotCount=3, entries [{100,1},{200,2}].
//
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v48 ida=0x534dc4
func TestOperationDiscardV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := OperationDiscard{
		count:          2,
		emptySlotCount: 3,
		entries: []DiscardEntry{
			{id: 100, flag: 1},
			{id: 200, flag: 2},
		},
	}
	want := []byte{
		0x02,                   // count = 2 (deleteCount Encode1 @0x534e9a)
		0x03,                   // emptySlotCount = 3 (Encode1 @0x534ea5)
		0x64, 0x00, 0x00, 0x00, // entry[0].id = 100 (Encode4 @0x534ed7)
		0x01,                   // entry[0].flag = 1 (Encode1 @0x534eec)
		0xC8, 0x00, 0x00, 0x00, // entry[1].id = 200 (Encode4 @0x534ed7)
		0x02, // entry[1].flag = 2 (Encode1 @0x534eec)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 NoteOperationDiscard golden mismatch\n got: % x\nwant: % x", got, want)
	}
}
