package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/note"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v95 ida=0x9f9da0
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v83 ida=0xa2508b
func TestNoteDisplayRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			notes := []note.NoteEntry{
				{Id: 1, SenderName: "Alice", Message: "Hello!", Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC), Flag: 1},
				{Id: 2, SenderName: "Bob", Message: "Hi there", Timestamp: time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC), Flag: 0},
			}
			input := NewNoteDisplay(3, notes)
			output := Display{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Notes()) != 2 {
				t.Fatalf("notes: got %d, want 2", len(output.Notes()))
			}
			if output.Notes()[0].Id != 1 {
				t.Errorf("note[0].Id: got %v, want 1", output.Notes()[0].Id)
			}
			if output.Notes()[0].SenderName != "Alice" {
				t.Errorf("note[0].SenderName: got %q, want %q", output.Notes()[0].SenderName, "Alice")
			}
			if output.Notes()[0].Message != "Hello!" {
				t.Errorf("note[0].Message: got %q, want %q", output.Notes()[0].Message, "Hello!")
			}
			if output.Notes()[0].Flag != 1 {
				t.Errorf("note[0].Flag: got %v, want 1", output.Notes()[0].Flag)
			}
			if output.Notes()[1].SenderName != "Bob" {
				t.Errorf("note[1].SenderName: got %q, want %q", output.Notes()[1].SenderName, "Bob")
			}
		})
	}
}

// TestNoteDisplayV79 pins the gms_v79 MEMO_RESULT (op 38) Display-mode wire.
//
// IDA-verified client decode (GMS_v79_1_DEVM.exe, port 13340) —
// CWvsContext::OnMemoResult @0x96f185, Display path (raw mode byte 3, since
// `Decode1(a2) - 3 == 0` @0x96f1a0):
//
//	Decode1 @0x96f26c → count (v12).
//	loop count× sub_4D8D86(a2) @0x96f284 → one GW_Memo each:
//	  Decode4    @0x4d8d98 → id.
//	  DecodeStr  @0x4d8da0 → sender (client lstrcpyA into this+4).
//	  DecodeStr  @0x4d8dc1 → message.
//	  DecodeBuffer(8) @0x4d8de1 → 8-byte FILETIME timestamp (opaque blob).
//	  Decode1    @0x4d8ded → flag.
//
// The Display-mode mode byte (3) is written first by atlas Display.Encode
// (WriteByte(mode), WriteByte(count), entries). atlas NoteEntry.Encode appends
// a trailing space to the sender (WriteAsciiString(sender+" ")) — the client
// lstrcpyA copies the string verbatim; the space is cosmetic padding.
// The 8-byte timestamp is the DecodeBuffer(8) opaque field; its bytes are
// derived from atlas model.MsTime (the FILETIME encoder) per the opaque-field
// discipline. Fixture: 2020-01-01T00:00:00Z → MsTime 132223104000000000.
//
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v79 ida=0x96f185
func TestNoteDisplayV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := NewNoteDisplay(3, []note.NoteEntry{
		{Id: 1, SenderName: "Alice", Message: "Hi", Timestamp: ts, Flag: 1},
	})
	want := []byte{
		0x03,                   // WriteByte mode = 3 (Display)
		0x01,                   // WriteByte count = 1
		0x01, 0x00, 0x00, 0x00, // Decode4 id = 1
		0x06, 0x00, 0x41, 0x6C, 0x69, 0x63, 0x65, 0x20, // DecodeStr sender "Alice " (len 6)
		0x02, 0x00, 0x48, 0x69, // DecodeStr message "Hi" (len 2)
		0x00, 0x00, 0x05, 0x69, 0x36, 0xC0, 0xD5, 0x01, // DecodeBuffer(8) FILETIME = MsTime(2020-01-01)
		0x01, // Decode1 flag = 1
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v79 NoteDisplay golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestNoteDisplayV72 pins the gms_v72 MEMO_RESULT Display-mode wire.
//
// IDA-verified client decode (GMS_v72.1_U_DEVM.exe, port 13339) —
// CWvsContext::OnMemoResult @0x91d23d, Display path (raw mode byte 3, since
// `Decode1(a2) - 3 == 0` @0x91d258):
//
//	Decode1 @0x91d324 → count (v12).
//	loop count× sub_4D0F8B(a2) @0x91d33c → one GW_Memo each:
//	  Decode4    @0x4d0f9d → id.
//	  DecodeStr  @0x4d0fa5 → sender (client lstrcpyA into this+4).
//	  DecodeStr  @0x4d0fc6 → message.
//	  DecodeBuffer(8) @0x4d0fe6 → 8-byte FILETIME timestamp (opaque blob).
//	  Decode1    @0x4d0ff2 → flag.
//
// Byte-identical to the verified v79 wire. The Display-mode mode byte (3) is
// written first by atlas Display.Encode. atlas NoteEntry.Encode appends a
// trailing space to the sender (WriteAsciiString(sender+" ")) — the client
// lstrcpyA copies verbatim; the space is cosmetic padding. The 8-byte timestamp
// is the DecodeBuffer(8) opaque field, derived from atlas model.MsTime per the
// opaque-field discipline. Fixture: 2020-01-01T00:00:00Z → MsTime 132223104000000000.
//
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v72 ida=0x91d23d
func TestNoteDisplayV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := NewNoteDisplay(3, []note.NoteEntry{
		{Id: 1, SenderName: "Alice", Message: "Hi", Timestamp: ts, Flag: 1},
	})
	want := []byte{
		0x03,                   // WriteByte mode = 3 (Display)
		0x01,                   // WriteByte count = 1 (@0x91d324)
		0x01, 0x00, 0x00, 0x00, // Decode4 id = 1 (@0x4d0f9d)
		0x06, 0x00, 0x41, 0x6C, 0x69, 0x63, 0x65, 0x20, // DecodeStr sender "Alice " (len 6) (@0x4d0fa5)
		0x02, 0x00, 0x48, 0x69, // DecodeStr message "Hi" (len 2) (@0x4d0fc6)
		0x00, 0x00, 0x05, 0x69, 0x36, 0xC0, 0xD5, 0x01, // DecodeBuffer(8) FILETIME = MsTime(2020-01-01) (@0x4d0fe6)
		0x01, // Decode1 flag = 1 (@0x4d0ff2)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v72 NoteDisplay golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestNoteDisplayV84 pins the gms_v84 MEMO_RESULT (op 41 / 0x029) Display-mode wire.
//
// IDA-verified client decode (GMS_v84.1_U_DEVM.i64, session 79511a2a) —
// CWvsContext::OnMemoResult @0xa70785:
//
//	Decode1 @0xa707a0 → mode; `mode - 3 == 0` selects the Display path
//	  (the else-branch at the bottom of the function; nonzero routes to the
//	  SendSuccess(4)/SendError(5) StringPool-notice paths or, at mode==7,
//	  tail-calls the unrelated CWvsContext::OnMemoNotify_Receive @0xa708ea).
//	sub_530434(this+3511) @0xa7085f → clears the existing memo list (not a
//	  wire read).
//	Decode1 @0xa7086c → count (v21).
//	loop count× :
//	  sub_5303F6(this+3511) @0xa7087d → append list node (not a wire read).
//	  sub_4EBF0C(a2) @0xa70884 → one GW_Memo entry:
//	    Decode4      @0x4ebf1e → id.
//	    DecodeStr    @0x4ebf26 → sender (lstrcpynA truncates the client-side
//	      copy to 13 bytes @0x4ebf39, but DecodeStr still consumes the full
//	      length-prefixed string off the wire — the wire format is
//	      unaffected).
//	    DecodeStr    @0x4ebf49 → message (same lstrcpynA(13) truncation note
//	      @0x4ebf56).
//	    DecodeBuffer(8) @0x4ebf6b → 8-byte FILETIME timestamp (opaque blob).
//	    Decode1      @0x4ebf77 → flag.
//
// Read order is byte-identical to the verified v79/v72/v83 Display wire.
// Opcode confirmed from the raw-opcode dispatch: CClientSocket::ProcessPacket
// @0x49b502 does `v2 = Decode2(iPacket)` @0x49b530 (the raw 2-byte wire
// opcode, no offset) and for 0x1D..0x7F tail-calls
// `CWvsContext::OnPacket(v2, iPacket)` @0x49b55d; CWvsContext::OnPacket
// @0xa51cd0 switches on that same `a1` and `case 0x29:` @0xa51ce2 calls
// `CWvsContext::OnMemoResult(Src)` @0xa51df0 — opcode 0x29 (41 decimal),
// matching registry gms_v84.yaml MEMO_RESULT opcode: 41.
//
// The Display-mode mode byte (3) is written first by atlas Display.Encode
// (WriteByte(mode), WriteByte(count), entries). atlas NoteEntry.Encode appends
// a trailing space to the sender (WriteAsciiString(sender+" ")) — the client
// DecodeStr/lstrcpynA copies verbatim (then truncates at 13 bytes internally);
// the space is cosmetic padding on the wire. The 8-byte timestamp is the
// DecodeBuffer(8) opaque field; its bytes are derived from atlas
// model.MsTime (the FILETIME encoder) per the opaque-field discipline.
// Fixture: 2020-01-01T00:00:00Z → MsTime 132223104000000000.
//
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v84 ida=0xa70785
func TestNoteDisplayV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := NewNoteDisplay(3, []note.NoteEntry{
		{Id: 1, SenderName: "Alice", Message: "Hi", Timestamp: ts, Flag: 1},
	})
	want := []byte{
		0x03,                   // WriteByte mode = 3 (Display) (@0xa707a0)
		0x01,                   // WriteByte count = 1 (@0xa7086c)
		0x01, 0x00, 0x00, 0x00, // Decode4 id = 1 (@0x4ebf1e)
		0x06, 0x00, 0x41, 0x6C, 0x69, 0x63, 0x65, 0x20, // DecodeStr sender "Alice " (len 6) (@0x4ebf26)
		0x02, 0x00, 0x48, 0x69, // DecodeStr message "Hi" (len 2) (@0x4ebf49)
		0x00, 0x00, 0x05, 0x69, 0x36, 0xC0, 0xD5, 0x01, // DecodeBuffer(8) FILETIME = MsTime(2020-01-01) (@0x4ebf6b)
		0x01, // Decode1 flag = 1 (@0x4ebf77)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v84 NoteDisplay golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestNoteDisplayJMS185 pins the jms_v185 MEMO_RESULT (op 0x026/38)
// Display-mode wire.
//
// IDA-verified client decode (MapleStory_dump_SCY.exe.i64, session 3c4bb8b1,
// the _SCY build) — CWvsContext::OnMemoResult @0xb0c6d0:
//
//	Decode1 @0xb0c6eb → mode; `v3 = mode - 3`, `v3 == 0` selects the Display
//	  path (the else-branch at the bottom of the function; nonzero routes to
//	  the SendSuccess(4)/SendError(5) StringPool-notice paths or, at mode==7,
//	  tail-calls CWvsContext::OnMemoNotify_Receive @0xb0c708/0xb0c700 — no
//	  further wire read).
//	sub_55C3E4(this+3511) @0xb0c7aa → clears the existing memo list (not a
//	  wire read).
//	Decode1 @0xb0c7b2 → count (v12/v23).
//	loop count× :
//	  sub_55C3A6(this+3511) @0xb0c7c8 → append list node (not a wire read).
//	  sub_510E70(v14, iPacket) @0xb0c7cf → one GW_Memo entry:
//	    Decode4      @0x510e82 → id.
//	    DecodeStr    @0x510e8a → sender.
//	    DecodeStr    @0x510ead → message.
//	    DecodeBuffer(8) @0x510ed2 → 8-byte FILETIME timestamp (opaque blob).
//	    Decode1      @0x510ede → flag.
//
// Read order is byte-identical to the verified v72/v79/v83/v84/v87/v95
// Display wire. Opcode confirmed end-to-end: CClientSocket::ProcessPacket
// @0x4b17eb does `v3 = Decode2(iPacket)` @0x4b1819 (the raw 2-byte wire
// opcode, no offset) and for 0x1B..0x7A tail-calls
// `CWvsContext::OnPacket(v3, iPacket)` @0x4b1891; CWvsContext::OnPacket
// @0xaebfe7 switches on that same value and `case 0x26:` @0xaec0fa calls
// `CWvsContext::OnMemoResult(this, iPacket)` — opcode 0x026 (38 decimal),
// matching registry jms_v185.yaml MEMO_RESULT opcode: 38.
//
// The Display-mode mode byte (3) is written first by atlas Display.Encode
// (WriteByte(mode), WriteByte(count), entries). atlas NoteEntry.Encode appends
// a trailing space to the sender (WriteAsciiString(sender+" ")) — the client
// DecodeStr copies verbatim; the space is cosmetic padding on the wire. The
// 8-byte timestamp is the DecodeBuffer(8) opaque field; its bytes are derived
// from atlas model.MsTime (the FILETIME encoder) per the opaque-field
// discipline. Fixture: 2020-01-01T00:00:00Z → MsTime 132223104000000000.
//
// packet-audit:verify packet=note/clientbound/NoteDisplay version=jms_v185 ida=0xb0c6d0
func TestNoteDisplayJMS185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := NewNoteDisplay(3, []note.NoteEntry{
		{Id: 1, SenderName: "Alice", Message: "Hi", Timestamp: ts, Flag: 1},
	})
	want := []byte{
		0x03,                   // WriteByte mode = 3 (Display) (@0xb0c6eb)
		0x01,                   // WriteByte count = 1 (@0xb0c7b2)
		0x01, 0x00, 0x00, 0x00, // Decode4 id = 1 (@0x510e82)
		0x06, 0x00, 0x41, 0x6C, 0x69, 0x63, 0x65, 0x20, // DecodeStr sender "Alice " (len 6) (@0x510e8a)
		0x02, 0x00, 0x48, 0x69, // DecodeStr message "Hi" (len 2) (@0x510ead)
		0x00, 0x00, 0x05, 0x69, 0x36, 0xC0, 0xD5, 0x01, // DecodeBuffer(8) FILETIME = MsTime(2020-01-01) (@0x510ed2)
		0x01, // Decode1 flag = 1 (@0x510ede)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("jms_v185 NoteDisplay golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestNoteDisplayEmptyRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewNoteDisplay(3, nil)
	output := Display{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Notes()) != 0 {
		t.Errorf("notes: got %d, want 0", len(output.Notes()))
	}
}
