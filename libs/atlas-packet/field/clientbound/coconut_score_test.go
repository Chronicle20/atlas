package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldCoconutScore version=gms_v83 ida=0x549802
// packet-audit:verify packet=field/clientbound/FieldCoconutScore version=gms_v84 ida=0x555f75
// packet-audit:verify packet=field/clientbound/FieldCoconutScore version=gms_v87 ida=0x5734b7
// packet-audit:verify packet=field/clientbound/FieldCoconutScore version=gms_v95 ida=0x54bf70
// packet-audit:verify packet=field/clientbound/FieldCoconutScore version=jms_v185 ida=0x589ad1
func TestCoconutScoreGolden(t *testing.T) {
	input := NewCoconutScore(0x000A, 0x0014)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x0A, 0x00, 0x14, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestCoconutScoreRoundTrip(t *testing.T) {
	input := NewCoconutScore(0x000A, 0x0014)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
