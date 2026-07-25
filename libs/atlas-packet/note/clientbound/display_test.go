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

func TestNoteDisplayEmptyRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewNoteDisplay(3, nil)
	output := Display{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Notes()) != 0 {
		t.Errorf("notes: got %d, want 0", len(output.Notes()))
	}
}
