package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// noteDiscardSpecialFlagJMS is jms_v185's special (gift) memo flag, used by the
// jms fixtures below. The production codec resolves the special flag per version
// via noteDiscardSpecialFlag(t); this test-local constant keeps the jms cases
// readable.
const noteDiscardSpecialFlagJMS = 3

// TestOperationDiscardPlainByteOutput pins the REAL client SetRet wire shape
// for a plain-note discard — a batch with NO "special" (gift/reward) memo —
// the common real-world case that CRASHED live clients when an earlier
// codec revision omitted the specialCount header byte for non-JMS versions
// (the decode desynced starting at the first entry).
//
// Re-decompiled during this verification pass (session ids in the marker
// list below; see also the fresh IDA output cited in v48_test.go/v61_test.go
// for the two legacy versions) to confirm CMemoListDlg::SetRet writes, on
// EVERY version without exception:
//
//	Encode1(1)             → mode byte; consumed by the NoteOperationHandle
//	                          dispatcher BEFORE this codec runs (not part of
//	                          the body below).
//	Encode1(totalCount)    → total local memo-list length.
//	Encode1(specialCount)  → # memos flagged as the version's special value
//	                          (noteDiscardSpecialFlag: 2 on GMS<=61, 3
//	                          otherwise), counted BEFORE the free-slot budget
//	                          is applied. Present on EVERY version — an
//	                          earlier revision wrongly modeled this as
//	                          jms-only, which mis-aligned the GMS body by one
//	                          byte and crashed the client on decode.
//	Encode1(emptySlotCount) → inbox ETC free-slot budget.
//	per memo: Encode4(id) + Encode1(flag) — flag values 0/1 used here are
//	          non-special on every version (special is 2 or 3), so no
//	          trailing extra field is written; see
//	          TestOperationDiscardJMSSpecialEntry and v48_test.go/v61_test.go
//	          for the special-entry (trailing extra int32) shape.
//
// Per-version confirmation (SetRet address, decompile session):
//
//	gms_v72 0x5fb443 (90e36cb0) · gms_v79 0x619f32 (9a7d3642)
//	gms_v83 0x64aa57 (ce4ff298) · gms_v84 0x6606a0 (79511a2a)
//	gms_v87 0x684843 (81f32170) · gms_v95 0x624280 (e4abcb98)
//	jms_v185 0x6c2d43 (3c4bb8b1, per TestOperationDiscardJMSSpecialEntry's
//	earlier from-scratch decompile)
//
// Because the header/entry shape for this plain case has no version-specific
// field, the golden bytes are IDENTICAL across all seven versions below (and
// v48/v61, whose own byte-golden fixtures live in v48_test.go/v61_test.go).
// Fixture: count=2, specialCount=0, emptySlotCount=3, entries
// [{id:100,flag:0},{id:200,flag:1}].
//
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v72 ida=0x5fb443
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v79 ida=0x619f32
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v83 ida=0x64aa57
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v84 ida=0x6606a0
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v87 ida=0x684843
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v95 ida=0x624280
//
// jms_v185 is exercised in this table too (below) but is NOT re-marked here
// — its packet-audit marker lives on TestOperationDiscardJMSSpecialEntry,
// which is the richer fixture (covers both the plain and special-entry
// shapes); a second marker for the same packet×version would be a duplicate
// (matrix --check fails on that).
func TestOperationDiscardPlainByteOutput(t *testing.T) {
	want := []byte{
		0x02,                   // count = 2 (totalCount)
		0x00,                   // specialCount = 0 (no gift/reward memo in this batch)
		0x03,                   // emptySlotCount = 3
		0x64, 0x00, 0x00, 0x00, // entry[0].id = 100
		0x00,                   // entry[0].flag = 0 (normal)
		0xC8, 0x00, 0x00, 0x00, // entry[1].id = 200
		0x01, // entry[1].flag = 1 (normal)
	}
	input := OperationDiscard{
		count:          2,
		specialCount:   0,
		emptySlotCount: 3,
		entries: []DiscardEntry{
			{id: 100, flag: 0},
			{id: 200, flag: 1},
		},
	}
	variants := []pt.TenantVariant{
		pt.Variants[9],  // GMS v72
		pt.Variants[10], // GMS v79
		pt.Variants[1],  // GMS v83
		pt.Variants[5],  // GMS v84
		pt.Variants[2],  // GMS v87
		pt.Variants[3],  // GMS v95
		pt.Variants[4],  // JMS v185
	}
	for _, v := range variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Errorf("%s NoteOperationDiscard golden mismatch\n got: % x\nwant: % x", v.Name, got, want)
			}
		})
	}
}

// TestOperationDiscardGMS72PlusSpecialByteOutput exercises the flag==3
// special-entry shape shared by gms_v72..gms_v95 (noteDiscardSpecialFlag
// returns 3 for GMS > 61): one trailing extra1 int32, IDA-confirmed absent
// a second extra field (unlike jms_v185's two, see
// TestOperationDiscardJMSSpecialEntry). Representative version: gms_v83
// (CMemoListDlg::SetRet@0x64aa57, session ce4ff298) — Encode4(id)@0x64abf8 +
// Encode1(flag)@0x64ac0d + Encode4(extra1)@0x64ac18; the same shape was
// re-confirmed on v72/v79/v84/v87/v95 during this pass (see
// TestOperationDiscardPlainByteOutput's session list).
func TestOperationDiscardGMS72PlusSpecialByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationDiscard{
		count:          2,
		specialCount:   1,
		emptySlotCount: 3,
		entries: []DiscardEntry{
			{id: 100, flag: 1},
			{id: 200, flag: 3, extra1: 500},
		},
	}
	want := []byte{
		0x02,                   // count = 2
		0x01,                   // specialCount = 1
		0x03,                   // emptySlotCount = 3
		0x64, 0x00, 0x00, 0x00, // entry[0].id = 100
		0x01,                   // entry[0].flag = 1 (normal)
		0xC8, 0x00, 0x00, 0x00, // entry[1].id = 200
		0x03,                   // entry[1].flag = 3 (special)
		0xF4, 0x01, 0x00, 0x00, // entry[1].extra1 = 500
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("gms_v83 NoteOperationDiscard special-entry golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestOperationDiscardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDiscard{
				count:          2,
				emptySlotCount: 3,
				entries: []DiscardEntry{
					{id: 100, flag: 1},
					{id: 200, flag: 2},
				},
			}
			output := OperationDiscard{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Count() != input.Count() {
				t.Errorf("count: got %v, want %v", output.Count(), input.Count())
			}
			if output.SpecialCount() != input.SpecialCount() {
				t.Errorf("specialCount: got %v, want %v", output.SpecialCount(), input.SpecialCount())
			}
			if output.EmptySlotCount() != input.EmptySlotCount() {
				t.Errorf("emptySlotCount: got %v, want %v", output.EmptySlotCount(), input.EmptySlotCount())
			}
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries length: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i, e := range output.Entries() {
				if e.Id() != input.Entries()[i].Id() {
					t.Errorf("entry[%d].id: got %v, want %v", i, e.Id(), input.Entries()[i].Id())
				}
				if e.Flag() != input.Entries()[i].Flag() {
					t.Errorf("entry[%d].flag: got %v, want %v", i, e.Flag(), input.Entries()[i].Flag())
				}
			}
		})
	}
}

// TestOperationDiscardJMSSpecialEntry verifies jms_v185's outlier
// CMemoListDlg::SetRet@0x6c2d43 shape, re-decompiled from scratch during this
// verification pass (session 3c4bb8b1, MapleStory_dump_SCY.exe.i64) rather
// than assumed from the GMS codec shape:
//
//	if (arg0 == 2 || arg0 == 1) if (CUtilDlg::YesNo(...) == 6) {
//	  COutPacket::COutPacket(v25, 0x86)                      @0x6c2dc8  (NOTE_ACTION opcode, matches STATUS.md jms185 0x086)
//	  COutPacket::Encode1(v25, 1u)                            @0x6c2dd5  mode=1 (DISCARD)
//	  COutPacket::Encode1(v25, v5 /* array length */)         @0x6c2def  totalCount
//	  COutPacket::Encode1(v25, nType /* flag==3 count */)     @0x6c2e1d  specialCount   <- extra header byte vs v83..v95
//	  COutPacket::Encode1(v25, n[0] /* sub_51F1E3(4) */)      @0x6c2e28  emptySlots
//	  for each of the totalCount entries (this+204, stride 28):
//	    if flag == 3 (special):
//	      if emptySlots budget exhausted (*n<=0)              @0x6c2e88  entry SKIPPED — no bytes at all
//	      else:
//	        Encode4(SN) + Encode1(flag=3)                      @0x6c2f81 / 0x6c2f99
//	        Encode4(extra1) + Encode4(extra2)                  @0x6c2fa4 / 0x6c2faf   <- TWO extra int32s vs v83..v95's one-field-less shape
//	        emptySlots--
//	    else (normal):
//	      Encode4(SN) + Encode1(flag)                          @0x6c2e63 / 0x6c2e7b
//	}
//
// Because a skipped special entry emits zero bytes, the number of entries
// actually on the wire is deterministic from the header alone:
// wireEntries = totalCount - max(0, specialCount - emptySlots) — every
// special entry that DOES reach the wire was, by construction, written while
// budget remained, so it always carries the two trailing extra fields; no
// running-budget bookkeeping is needed on decode (see Decode's comment).
//
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=jms_v185 ida=0x6c2d43
func TestOperationDiscardJMSSpecialEntry(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)

	t.Run("no skip - special entry has room", func(t *testing.T) {
		// 3 local entries: 2 normal + 1 special(flag=3), 1 free slot -> nothing skipped.
		input := OperationDiscard{
			count:          3,
			specialCount:   1,
			emptySlotCount: 1,
			entries: []DiscardEntry{
				{id: 100, flag: 0},
				{id: 200, flag: 0},
				{id: 300, flag: noteDiscardSpecialFlagJMS, extra1: 999, extra2: 555},
			},
		}
		output := OperationDiscard{}
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

		if output.Count() != 3 {
			t.Errorf("count: got %v, want 3", output.Count())
		}
		if output.SpecialCount() != 1 {
			t.Errorf("specialCount: got %v, want 1", output.SpecialCount())
		}
		if output.EmptySlotCount() != 1 {
			t.Errorf("emptySlotCount: got %v, want 1", output.EmptySlotCount())
		}
		if len(output.Entries()) != 3 {
			t.Fatalf("entries length: got %v, want 3", len(output.Entries()))
		}
		last := output.Entries()[2]
		if last.Id() != 300 || last.Flag() != noteDiscardSpecialFlagJMS {
			t.Errorf("special entry id/flag: got id=%v flag=%v, want id=300 flag=%v", last.Id(), last.Flag(), noteDiscardSpecialFlagJMS)
		}
		if last.Extra1() != 999 || last.Extra2() != 555 {
			t.Errorf("special entry extras: got (%v,%v), want (999,555)", last.Extra1(), last.Extra2())
		}
		for i := 0; i < 2; i++ {
			if output.Entries()[i].Flag() != 0 || (output.Entries()[i].Extra1() != 0 || output.Entries()[i].Extra2() != 0) {
				t.Errorf("normal entry[%d] carries unexpected extras: %+v", i, output.Entries()[i])
			}
		}
	})

	t.Run("skip - special entry exhausts budget (0x6c2e88)", func(t *testing.T) {
		// 3 local entries: 1 normal + 2 special(flag=3), 0 free slots -> both
		// specials skipped entirely (no bytes for them at all), so the wire
		// carries just the 1 normal entry. wireEntries = 3 - max(0, 2-0) = 1.
		input := OperationDiscard{
			count:          3,
			specialCount:   2,
			emptySlotCount: 0,
			entries: []DiscardEntry{
				{id: 400, flag: 0},
			},
		}
		output := OperationDiscard{}
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

		if output.Count() != 3 {
			t.Errorf("count: got %v, want 3", output.Count())
		}
		if output.SpecialCount() != 2 {
			t.Errorf("specialCount: got %v, want 2", output.SpecialCount())
		}
		if len(output.Entries()) != 1 {
			t.Fatalf("entries length: got %v, want 1 (both special entries skipped, budget exhausted)", len(output.Entries()))
		}
		if output.Entries()[0].Id() != 400 || output.Entries()[0].Flag() != 0 {
			t.Errorf("surviving normal entry: got %+v, want id=400 flag=0", output.Entries()[0])
		}
	})

	t.Run("partial skip - budget covers only one of two specials", func(t *testing.T) {
		// 2 special entries, 1 free slot: the FIRST special encountered in
		// array order consumes the only slot; the second is skipped.
		// wireEntries = 2 - max(0, 2-1) = 1.
		input := OperationDiscard{
			count:          2,
			specialCount:   2,
			emptySlotCount: 1,
			entries: []DiscardEntry{
				{id: 500, flag: noteDiscardSpecialFlagJMS, extra1: 7, extra2: 8},
			},
		}
		output := OperationDiscard{}
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

		if len(output.Entries()) != 1 {
			t.Fatalf("entries length: got %v, want 1", len(output.Entries()))
		}
		e := output.Entries()[0]
		if e.Id() != 500 || e.Flag() != noteDiscardSpecialFlagJMS || e.Extra1() != 7 || e.Extra2() != 8 {
			t.Errorf("surviving special entry: got %+v, want id=500 flag=%v extra1=7 extra2=8", e, noteDiscardSpecialFlagJMS)
		}
	})
}
