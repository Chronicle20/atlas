package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNoteOperationArmsV79 pins the gms_v79 MEMO_RESULT (op 38) non-Display
// arms. IDA-verified — CWvsContext::OnMemoResult @0x96f185, switch on
// (Decode1(mode) - 3) @0x96f1a0:
//
//	raw mode 4 (v3==1) @0x96f246 → StringPool(2695)+Notice, NO further read
//	  → SendSuccess, mode-only.
//	raw mode 5 (v3==2) @0x96f1eb → Decode1(errorCode) then 0/1/2 → StringPool
//	  2636/2637/2638 → SendError, mode + 1 errorCode byte.
//	raw mode 6 (v3==3) @0x96f1b5 → return, NO read → Refresh, mode-only.
//
// v79 mode bytes are identical to v83 (SendSuccess=4, SendError=5, Refresh=6).
//
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v79 ida=0x96f185
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v79 ida=0x96f185
// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v79 ida=0x96f185
func TestNoteOperationArmsV79(t *testing.T) {
	if got := NewNoteSendSuccess(4).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x04}) {
		t.Errorf("v79 NoteSendSuccess: got % x want 04", got)
	}
	if got := NewNoteSendError(5, 1).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x05, 0x01}) {
		t.Errorf("v79 NoteSendError: got % x want 05 01", got)
	}
	if got := NewNoteRefresh(6).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x06}) {
		t.Errorf("v79 NoteRefresh: got % x want 06", got)
	}
}

// TestNoteOperationArmsV72 pins the gms_v72 MEMO_RESULT non-Display arms.
// IDA-verified — CWvsContext::OnMemoResult @0x91d23d, switch on
// (Decode1(mode) - 3) @0x91d258:
//
//	raw mode 4 (v5==0) @0x91d2fe → else-block StringPool(2693)+Notice, NO
//	  further read → SendSuccess, mode-only.
//	raw mode 5 (v6==0) → Decode1(errorCode) @0x91d2a3 then 0/1/2 → StringPool
//	  2634/2635/2636 → SendError, mode + 1 errorCode byte.
//	raw mode 6 (v6==1) @0x91d26d → return, NO read → Refresh, mode-only.
//	  (raw mode 7 (v6==2) → OnMemoNotify_Receive, out of scope.)
//
// v72 mode bytes are identical to v79/v83 (SendSuccess=4, SendError=5,
// Refresh=6); only the UI StringPool ids differ (not on the wire).
//
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v72 ida=0x91d23d
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v72 ida=0x91d23d
// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v72 ida=0x91d23d
func TestNoteOperationArmsV72(t *testing.T) {
	if got := NewNoteSendSuccess(4).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x04}) {
		t.Errorf("v72 NoteSendSuccess: got % x want 04", got)
	}
	if got := NewNoteSendError(5, 1).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x05, 0x01}) {
		t.Errorf("v72 NoteSendError: got % x want 05 01", got)
	}
	if got := NewNoteRefresh(6).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x06}) {
		t.Errorf("v72 NoteRefresh: got % x want 06", got)
	}
}

// TestNoteOperationArmsV84 pins the gms_v84 MEMO_RESULT (op 41 / 0x029)
// non-Display arms. IDA-verified — CWvsContext::OnMemoResult @0xa70785,
// switch on (Decode1(mode) - 3) @0xa707a0:
//
//	raw mode 4 (v6==0) @0xa707aa → else-block StringPool(2714)+Notice, NO
//	  further read @0xa7080a-0xa70814 → SendSuccess, mode-only.
//	raw mode 5 (v7==0) @0xa707b1 → Decode1(errorCode) @0xa707eb then 0/1/2 →
//	  StringPool 2652/2653/2654 → SendError, mode + 1 errorCode byte.
//	raw mode 7 (result==0, result=v7-2) @0xa707b4 → tail-calls
//	  CWvsContext::OnMemoNotify_Receive(this) @0xa707bd/0xa708ea with no
//	  CInPacket arg — NO further wire read → Refresh, mode-only.
//
// v84's Refresh mode number (7) diverges from v79/v72/v83/v87/v95 (6/8) —
// SendSuccess(4)/SendError(5) are unchanged. Wire SHAPE for every arm
// (mode-only / mode+errorCode / mode-only) is identical across all versions.
//
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v84 ida=0xa70785
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v84 ida=0xa70785
// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v84 ida=0xa70785
func TestNoteOperationArmsV84(t *testing.T) {
	if got := NewNoteSendSuccess(4).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x04}) {
		t.Errorf("v84 NoteSendSuccess: got % x want 04", got)
	}
	if got := NewNoteSendError(5, 1).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x05, 0x01}) {
		t.Errorf("v84 NoteSendError: got % x want 05 01", got)
	}
	if got := NewNoteRefresh(7).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{0x07}) {
		t.Errorf("v84 NoteRefresh: got % x want 07", got)
	}
}

// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v95 ida=0x9f9da0
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v95 ida=0x9f9da0
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v95 ida=0x9f9da0
// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v83 ida=0xa2508b
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v83 ida=0xa2508b
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v83 ida=0xa2508b
func TestSendSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SendSuccess{mode: 4}
			output := SendSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

func TestSendErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SendError{mode: 5, errorCode: 1}
			output := SendError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ErrorCode() != input.ErrorCode() {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), input.ErrorCode())
			}
		})
	}
}

func TestRefreshRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Refresh{mode: 6}
			output := Refresh{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
