package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/note"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNoteDisplayV61Body pins the gms_v61 MEMO_RESULT Display-mode wire.
//
// IDA-verified client decode (GMS_v61.1_U_DEVM.exe, port 13338) —
// CWvsContext::OnMemoResult @0x8468be, Display path (raw mode byte 2, since
// `Decode1(a2) - 2 == 0` @0x8468da):
//
//	ZList<GW_Memo>::RemoveAll @0x846985 (clears the list, reads nothing).
//	Decode1 @0x846992 → count (v11).
//	loop count× sub_4B59FD(a2) @0x8469aa → one GW_Memo each:
//	  Decode4    @0x4b5a0f → id.
//	  DecodeStr  @0x4b5a17 → sender (client lstrcpyA into this+4).
//	  DecodeStr  @0x4b5a38 → message (lstrcpyA into this+17).
//	  DecodeBuffer(8) @0x4b5a58 → 8-byte FILETIME timestamp (opaque blob).
//	  Decode1    @0x4b5a64 → flag (this+226).
//
// The per-entry read order is byte-identical to the verified v72 GW_Memo
// decoder (sub_4D0F8B, TestNoteDisplayV72); only the leading Display mode byte
// differs (2 in v61 vs 3 in v72 — see the -1 mode-table shift verified in
// v61_mode_gate_test.go). atlas Display.Encode writes WriteByte(mode),
// WriteByte(count), entries; NoteEntry.Encode appends a trailing space to the
// sender (WriteAsciiString(sender+" ")) — the client lstrcpyA copies verbatim,
// the space is cosmetic padding. The 8-byte timestamp is the DecodeBuffer(8)
// opaque field derived from atlas model.MsTime per the opaque-field discipline.
// Fixture: 2020-01-01T00:00:00Z → MsTime 132223104000000000.
//
// packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v61 ida=0x8468be
func TestNoteDisplayV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := NewNoteDisplay(2, []note.NoteEntry{
		{Id: 1, SenderName: "Alice", Message: "Hi", Timestamp: ts, Flag: 1},
	})
	want := []byte{
		0x02,                   // WriteByte mode = 2 (Display, Decode1-2==0 @0x8468da)
		0x01,                   // WriteByte count = 1 (@0x846992)
		0x01, 0x00, 0x00, 0x00, // Decode4 id = 1 (@0x4b5a0f)
		0x06, 0x00, 0x41, 0x6C, 0x69, 0x63, 0x65, 0x20, // DecodeStr sender "Alice " (len 6) (@0x4b5a17)
		0x02, 0x00, 0x48, 0x69, // DecodeStr message "Hi" (len 2) (@0x4b5a38)
		0x00, 0x00, 0x05, 0x69, 0x36, 0xC0, 0xD5, 0x01, // DecodeBuffer(8) FILETIME = MsTime(2020-01-01) (@0x4b5a58)
		0x01, // Decode1 flag = 1 (@0x4b5a64)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 NoteDisplay golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestNoteOperationArmsV61 pins the gms_v61 MEMO_RESULT non-Display arms.
// IDA-verified — CWvsContext::OnMemoResult @0x8468be, switch on
// (Decode1(mode) - 2) @0x8468da:
//
//	raw mode 3 (v3==1, v5==0) @0x84696c → else-block Notice(2652), NO further
//	  read → SendSuccess, mode-only.
//	raw mode 4 (v3==1, v5==1) @0x846913 → Decode1(errorCode) then 0/1/2 →
//	  Notice 2598/2599/2600 → SendError, mode + 1 errorCode byte.
//	raw mode >=5 @0x8468e9 → return, NO read. v61 has NO OnMemoNotify_Receive
//	  arm — the v72 REFRESH mode (6/7) is absent, dispositioned n-a.
//
// v61 mode bytes are the v72 values minus one (SendSuccess 4→3, SendError
// 5→4); only the UI StringPool ids differ (not on the wire). The atlas
// SendSuccess/SendError codecs are version-agnostic (WriteByte(mode) [+ error
// byte]); the mode value is supplied explicitly here, traced to the decompile.
//
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v61 ida=0x8468be
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v61 ida=0x8468be
func TestNoteOperationArmsV61(t *testing.T) {
	if got := NewNoteSendSuccess(3).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x03}) {
		t.Errorf("v61 NoteSendSuccess: got % x want 03", got)
	}
	if got := NewNoteSendError(4, 1).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x04, 0x01}) {
		t.Errorf("v61 NoteSendError: got % x want 04 01", got)
	}
}
