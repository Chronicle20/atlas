package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSnowballTouch version=gms_v79 ida=0x55288e
// packet-audit:verify packet=field/clientbound/FieldSnowballTouch version=gms_v83 ida=0x575372
// packet-audit:verify packet=field/clientbound/FieldSnowballTouch version=gms_v84 ida=0x584ceb
// packet-audit:verify packet=field/clientbound/FieldSnowballTouch version=gms_v87 ida=0x5a35f7
// packet-audit:verify packet=field/clientbound/FieldSnowballTouch version=gms_v95 ida=0x560510
// packet-audit:verify packet=field/clientbound/FieldSnowballTouch version=jms_v185 ida=0x5c986c
func TestSnowballTouchGolden(t *testing.T) {
	input := NewSnowballTouch()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

// TestSnowballTouchByteOutputV79 pins the gms_v79 FIELD_SNOWBALL_TOUCH
// clientbound read. IDA: CField_Snowball::OnTouch @0x55288e
// (GMS_v79_1_DEVM.exe) reads no fields. Body is byte-identical (empty) to
// the v83 golden.
func TestSnowballTouchByteOutputV79(t *testing.T) {
	input := NewSnowballTouch()
	ctx := test.CreateContext("GMS", 79, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v79 golden mismatch: got %v want empty", actual)
	}
}

func TestSnowballTouchRoundTrip(t *testing.T) {
	input := NewSnowballTouch()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
