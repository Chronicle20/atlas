package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v83 ida=0x53e5e1
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v84 ida=0x54abaa
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v87 ida=0x567b7d
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v95 ida=0x5492b0
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=jms_v185 ida=0x57dc4a
func TestAriantArenaUserScoreGolden(t *testing.T) {
	input := NewAriantArenaUserScore(0x01, "AB", 0x00000064)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02, 0x00, 0x41, 0x42, 0x64, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAriantArenaUserScoreRoundTrip(t *testing.T) {
	input := NewAriantArenaUserScore(0x01, "AB", 0x00000064)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
