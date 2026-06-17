package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// FieldMtsOperation is the #Mode synthetic cell of CITC::OnNormalItemResult
// (MTS_OPERATION): the OP-MODE-PREFIX contract is the single leading mode byte
// the dispatcher reads via Decode1 before switch-dispatching. The mode-only
// MtsOperation struct that originally backed this cell was retired (its
// mode-byte-only Encode was a false-pass once the per-mode body codecs landed).
// Each notice-only arm now has its OWN discrete per-mode struct that fixes its
// own mode byte (mts_result_empty_modes.go); the verified single-mode-byte wire
// contract this #Mode cell documents is demonstrated below by one of those
// discrete arms (RegisterSaleEntryDone, 0x1D). These markers keep the #Mode cell
// linked to its per-version evidence (the dispatcher Decode1 addresses are
// version-stable, IDA-confirmed).
//
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v83 ida=0x5a4311
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v84 ida=0x5b47c8
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v87 ida=0x5d43d0
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v95 ida=0x5771d0
func TestMtsOperationModeGolden(t *testing.T) {
	// OP-MODE-PREFIX: each per-mode codec owns its leading mode byte and stops
	// (for the Empty-shape arms). RegisterSaleEntryDone fixes mode 0x1D.
	input := NewMtsResultRegisterSaleEntryDone()
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x1D}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}
