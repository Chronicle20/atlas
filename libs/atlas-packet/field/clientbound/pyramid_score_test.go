package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldPyramidScore version=gms_v83 ida=0x5617c5
// packet-audit:verify packet=field/clientbound/FieldPyramidScore version=gms_v84 ida=0x56e400
// packet-audit:verify packet=field/clientbound/FieldPyramidScore version=gms_v87 ida=0x58c2e7
// packet-audit:verify packet=field/clientbound/FieldPyramidScore version=gms_v95 ida=0x5596c0
// packet-audit:verify packet=field/clientbound/FieldPyramidScore version=jms_v185 ida=0x5abf79
func TestPyramidScoreGolden(t *testing.T) {
	input := NewPyramidScore(0x01, 0x000003E8)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0xE8, 0x03, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestPyramidScoreRoundTrip(t *testing.T) {
	input := NewPyramidScore(0x01, 0x000003E8)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
