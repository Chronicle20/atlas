package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// FieldMtsOperation is the #Mode synthetic cell of CITC::OnNormalItemResult
// (MTS_OPERATION): the OP-MODE-PREFIX contract is the single leading mode byte
// the dispatcher reads via Decode1 before switch-dispatching. The mode-only
// MtsOperation struct that originally backed this cell was retired in task-096
// (its mode-byte-only Encode was a false-pass once the per-mode body codecs
// landed); the verified wire output — exactly the mode byte — is now produced by
// MtsResultEmpty, the notice-only arm codec whose Encode writes the mode byte and
// stops. These markers keep the #Mode cell linked to its fresh per-version
// evidence (the dispatcher Decode1 addresses are version-stable, IDA-confirmed).
//
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v83 ida=0x5a4311
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v84 ida=0x5b47c8
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v87 ida=0x5d43d0
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v95 ida=0x5771d0
func TestMtsOperationModeGolden(t *testing.T) {
	// OP-MODE-PREFIX: the codec owns only the leading mode byte. mode 0x15 =
	// OnGetITCListDone (the first dispatcher arm).
	input := NewMtsResultEmpty(0x15)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x15}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMtsOperationModeRoundTrip(t *testing.T) {
	input := NewMtsResultEmpty(0x33)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultEmpty{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
