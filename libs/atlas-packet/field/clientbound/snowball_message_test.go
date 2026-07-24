package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSnowballMessage version=gms_v79 ida=0x5526e8
// packet-audit:verify packet=field/clientbound/FieldSnowballMessage version=gms_v83 ida=0x5751cc
// packet-audit:verify packet=field/clientbound/FieldSnowballMessage version=gms_v84 ida=0x584b45
// packet-audit:verify packet=field/clientbound/FieldSnowballMessage version=gms_v87 ida=0x5a3451
// packet-audit:verify packet=field/clientbound/FieldSnowballMessage version=gms_v95 ida=0x562040
// packet-audit:verify packet=field/clientbound/FieldSnowballMessage version=jms_v185 ida=0x5c96c6
func TestSnowballMessageGolden(t *testing.T) {
	input := NewSnowballMessage(0x01, 0x02)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSnowballMessageByteOutputV79 pins the gms_v79 FIELD_SNOWBALL_MESSAGE
// clientbound read. IDA: CField_Snowball::OnMessage @0x5526e8
// (GMS_v79_1_DEVM.exe). Body is byte-identical to the v83 golden.
func TestSnowballMessageByteOutputV79(t *testing.T) {
	input := NewSnowballMessage(0x01, 0x02)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x01, 0x02}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSnowballMessageRoundTrip(t *testing.T) {
	input := NewSnowballMessage(0x01, 0x02)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
