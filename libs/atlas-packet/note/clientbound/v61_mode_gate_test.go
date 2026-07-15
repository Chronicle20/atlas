package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNoteModeTableV61 proves the gms_v61 MEMO_RESULT (op 0x26/38) mode table
// is shifted -1 versus v72+ and that the *-Body encoders resolve the v61 mode
// bytes through the tenant `operations` table.
//
// IDA-verified — CWvsContext::OnMemoResult @0x8468be (GMS_v61.1_U_DEVM.exe,
// port 13338). The dispatch is `v3 = Decode1(mode) - 2` @0x8468da:
//
//	mode 2 (v3==0)            @0x846985 → Display  (RemoveAll + count + entries).
//	mode 3 (v3==1, v5==0)     @0x84696c → SendSuccess (Notice 2652, no read).
//	mode 4 (v3==1, v5==1)     @0x846905 → SendError  (Decode1 errorCode 0/1/2).
//	mode >=5                  @0x8468e9 → return, no-op (NO refresh/notify arm).
//
// This is a uniform -1 shift of the v72 table (SHOW 3→2, SEND_SUCCESS 4→3,
// SEND_ERROR 5→4) — v72 OnMemoResult @0x91d23d dispatches on `Decode1 - 3`.
// v61 additionally has NO OnMemoNotify_Receive / empty-return REFRESH arm
// (v72 modes 6/7), so the v61 seed template drops the REFRESH key entirely
// (dispositioned n-a). The template `operations` table now reads
// {SHOW:2, SEND_SUCCESS:3, SEND_ERROR:4}; this test pins that those values
// flow to the wire mode byte through ResolveCode.
func TestNoteModeTableV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)

	// Mirror of the corrected template_gms_61_1.json writer[25] operations table.
	ops := map[string]interface{}{
		"operations": map[string]interface{}{
			"SHOW":         float64(2),
			"SEND_SUCCESS": float64(3),
			"SEND_ERROR":   float64(4),
		},
	}

	// SHOW → Display mode byte 2 (Decode1-2==0 @0x8468da). Empty note list so
	// the wire is just [mode][count=0].
	t.Run("show", func(t *testing.T) {
		got := pt.Encode(t, ctx, NoteDisplayBody(nil), ops)
		if !bytes.Equal(got, []byte{0x02, 0x00}) {
			t.Errorf("v61 SHOW mode: got % x want 02 00", got)
		}
	})

	// SEND_SUCCESS → mode byte 3 (v5==0 @0x84696c, no further read).
	t.Run("send_success", func(t *testing.T) {
		got := pt.Encode(t, ctx, NoteSendSuccessBody(), ops)
		if !bytes.Equal(got, []byte{0x03}) {
			t.Errorf("v61 SEND_SUCCESS mode: got % x want 03", got)
		}
	})
}
