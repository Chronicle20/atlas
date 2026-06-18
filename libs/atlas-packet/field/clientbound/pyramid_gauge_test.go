package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldPyramidGauge version=gms_v83 ida=0x560a81
// packet-audit:verify packet=field/clientbound/FieldPyramidGauge version=gms_v84 ida=0x56d6bc
// packet-audit:verify packet=field/clientbound/FieldPyramidGauge version=gms_v87 ida=0x58b5a3
// packet-audit:verify packet=field/clientbound/FieldPyramidGauge version=gms_v95 ida=0x556200
// packet-audit:verify packet=field/clientbound/FieldPyramidGauge version=jms_v185 ida=0x5ab235
func TestPyramidGaugeGolden(t *testing.T) {
	input := NewPyramidGauge(0x00000064)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x64, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestPyramidGaugeRoundTrip(t *testing.T) {
	input := NewPyramidGauge(0x00000064)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
