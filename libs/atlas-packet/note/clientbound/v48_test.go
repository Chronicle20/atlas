package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/note"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNoteDisplayV48Body pins the gms_v48 MEMO_RESULT Display-mode wire.
//
// IDA-verified client decode (GMS_v48_1_DEVM.exe, port 13337) —
// CWvsContext::OnMemoResult @0x71d8e2, Display path. The dispatcher computes
// v3 = Decode1(a2) - 2 @0x71d8fe; the Display (list) arm is the v3==0 else-block
// @0x71d9a9 (i.e. raw sub-op byte 2 — identical to the verified v61 Display mode):
//
//	Decode1 @0x71d9a9 → count (v11).
//	loop count× sub_49CCDB(a2) @0x71d9c7 → one GW_Memo each:
//	  Decode4        @0x49cced → id.
//	  DecodeStr      @0x49ccf5 → sender (lstrcpyA into this+4).
//	  DecodeStr      @0x49cd16 → message (lstrcpyA into this+17).
//	  DecodeBuffer(8) @0x49cd33 → 8-byte FILETIME timestamp (opaque blob).
//	  Decode1        @0x49cd3f → flag (this+126).
//
// The per-entry read order and the Display mode byte (2) are byte-identical to
// the verified gms_v61 GW_Memo decoder (see TestNoteDisplayV61Body); the codec
// note/clientbound/display.go is version-agnostic, so the v48 wire is identical
// to v61. NoteEntry.Encode appends a trailing space to the sender
// (WriteAsciiString(sender+" ")) — the client lstrcpyA copies verbatim, the
// space is cosmetic padding. The 8-byte timestamp is the DecodeBuffer(8) opaque
// field derived from atlas model.MsTime per the opaque-field discipline.
// Fixture: 2020-01-01T00:00:00Z → MsTime 132223104000000000.
//
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v48 ida=0x71d8e2
func TestNoteDisplayV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := NewNoteDisplay(2, []note.NoteEntry{
		{Id: 1, SenderName: "Alice", Message: "Hi", Timestamp: ts, Flag: 1},
	})
	want := []byte{
		0x02,                   // WriteByte mode = 2 (Display, Decode1-2==0 @0x71d8fe)
		0x01,                   // WriteByte count = 1 (@0x71d9a9)
		0x01, 0x00, 0x00, 0x00, // Decode4 id = 1 (@0x49cced)
		0x06, 0x00, 0x41, 0x6C, 0x69, 0x63, 0x65, 0x20, // DecodeStr sender "Alice " (len 6) (@0x49ccf5)
		0x02, 0x00, 0x48, 0x69, // DecodeStr message "Hi" (len 2) (@0x49cd16)
		0x00, 0x00, 0x05, 0x69, 0x36, 0xC0, 0xD5, 0x01, // DecodeBuffer(8) FILETIME = MsTime(2020-01-01) (@0x49cd33)
		0x01, // Decode1 flag = 1 (@0x49cd3f)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 NoteDisplay golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestNoteOperationArmsV48 pins the gms_v48 MEMO_RESULT non-Display arms.
// IDA-verified — CWvsContext::OnMemoResult @0x71d8e2, dispatch on
// v3 = (Decode1(mode) - 2) @0x71d8fe:
//
//	raw mode 3 (v3==1, v5==0) @0x71d99f → Notice(2425), NO further read →
//	  SendSuccess, mode-only.
//	raw mode 4 (v3==2) @0x71d937 → Decode1(errorCode) then 0/1/2 → Notice
//	  2372/2373/2374 → SendError, mode + 1 errorCode byte.
//	raw mode >=5 @0x71d90d → return, NO read. v48 has NO Refresh arm (v72+'s
//	  REFRESH mode is absent) — the switch falls through to return.
//
// The v48 mode bytes (SendSuccess 3, SendError 4) are identical to the verified
// gms_v61 values (see TestNoteOperationArmsV61); only the UI StringPool ids
// differ (not on the wire). The atlas SendSuccess/SendError codecs are
// version-agnostic (WriteByte(mode) [+ error byte]); the mode value is supplied
// explicitly here, traced to the decompile.
//
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v48 ida=0x71d8e2
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v48 ida=0x71d8e2
func TestNoteOperationArmsV48(t *testing.T) {
	if got := NewNoteSendSuccess(3).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x03}) {
		t.Errorf("v48 NoteSendSuccess: got % x want 03", got)
	}
	if got := NewNoteSendError(4, 1).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x04, 0x01}) {
		t.Errorf("v48 NoteSendError: got % x want 04 01", got)
	}
}
