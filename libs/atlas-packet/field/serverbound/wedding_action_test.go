package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v83 ida=0x58153d
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v84 ida=0x5911e6
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v87 ida=0x5b012e
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v95 ida=0x5640f0
func TestWeddingActionGolden(t *testing.T) {
	input := NewWeddingAction(0x02)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestWeddingActionRoundTrip(t *testing.T) {
	input := NewWeddingAction(0x02)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := WeddingAction{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Step() != input.Step() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
