package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldAriantScore version=gms_v95 ida=0x564ad0
func TestAriantScoreGolden(t *testing.T) {
	input := NewAriantScore(0x09)
	ctx := test.CreateContext("GMS", 95, 1)
	expected := []byte{0x09}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAriantScoreRoundTrip(t *testing.T) {
	input := NewAriantScore(0x09)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
