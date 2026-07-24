package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldAriantArenaShowResult version=gms_v79 ida=0x52914d
// packet-audit:verify packet=field/clientbound/FieldAriantArenaShowResult version=gms_v83 ida=0x53ef95
// packet-audit:verify packet=field/clientbound/FieldAriantArenaShowResult version=gms_v84 ida=0x54b55e
// packet-audit:verify packet=field/clientbound/FieldAriantArenaShowResult version=gms_v87 ida=0x568553
// packet-audit:verify packet=field/clientbound/FieldAriantArenaShowResult version=gms_v95 ida=0x547630
// packet-audit:verify packet=field/clientbound/FieldAriantArenaShowResult version=jms_v185 ida=0x57e620
func TestAriantArenaShowResultGolden(t *testing.T) {
	input := NewAriantArenaShowResult()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

// TestAriantArenaShowResultByteOutputV79 pins the gms_v79
// FIELD_ARIANT_ARENA_SHOW_RESULT clientbound read. IDA:
// CField_AriantArena::OnShowResult @0x52914d (GMS_v79_1_DEVM.exe) reads no
// fields. Body is byte-identical (empty) to the v83 golden.
func TestAriantArenaShowResultByteOutputV79(t *testing.T) {
	input := NewAriantArenaShowResult()
	ctx := test.CreateContext("GMS", 79, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v79 golden mismatch: got %v want empty", actual)
	}
}

func TestAriantArenaShowResultRoundTrip(t *testing.T) {
	input := NewAriantArenaShowResult()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
