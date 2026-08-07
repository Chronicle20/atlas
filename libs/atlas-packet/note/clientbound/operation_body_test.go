package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNoteSendErrorBodyUsesSendErrorMode pins that the error body resolves
// the SEND_ERROR operations key (mode 5 here — the GMS v83 value; v48/v61
// tenants resolve 4), not SEND_SUCCESS. Client (CWvsContext::OnMemoResult):
// the SEND_ERROR arm clears the exclusive-request lock then reads one error
// byte, in all nine versions.
func TestNoteSendErrorBodyUsesSendErrorMode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"SEND_SUCCESS": float64(4),
			"SEND_ERROR":   float64(5),
		},
		"errors": map[string]interface{}{
			"RECEIVER_UNKNOWN": float64(1),
			"NO_NOTE_ITEM":     float64(3),
		},
	}
	l, _ := testlog.NewNullLogger()

	got := NoteSendErrorBody(NoteSendErrorNoNoteItem)(l, ctx)(options)
	want := []byte{0x05, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("NO_NOTE_ITEM body: got %v, want %v", got, want)
	}

	got = NoteSendErrorBody(NoteSendErrorReceiverUnknown)(l, ctx)(options)
	want = []byte{0x05, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("RECEIVER_UNKNOWN body: got %v, want %v", got, want)
	}
}
