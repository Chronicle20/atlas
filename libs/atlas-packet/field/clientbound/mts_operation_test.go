package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v83 ida=0x5a4311
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v84 ida=0x5b47c8
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v87 ida=0x5d43d0
// packet-audit:verify packet=field/clientbound/FieldMtsOperation version=gms_v95 ida=0x5771d0
func TestMtsOperationGolden(t *testing.T) {
	// OP-MODE-PREFIX: CITC::OnNormalItemResult reads Decode1(mode) then
	// switch-dispatches; the codec owns only the mode byte. mode 0x15 =
	// OnGetITCListDone (first arm).
	input := NewMtsOperation(0x15)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x15}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMtsOperationRoundTrip(t *testing.T) {
	input := NewMtsOperation(0x33)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsOperation{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
