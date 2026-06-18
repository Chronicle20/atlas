package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSheepRanchClothes version=gms_v83 ida=0x545c52
// packet-audit:verify packet=field/clientbound/FieldSheepRanchClothes version=gms_v84 ida=0x552303
// packet-audit:verify packet=field/clientbound/FieldSheepRanchClothes version=gms_v87 ida=0x56f6c2
// packet-audit:verify packet=field/clientbound/FieldSheepRanchClothes version=gms_v95 ida=0x5499e0
// packet-audit:verify packet=field/clientbound/FieldSheepRanchClothes version=jms_v185 ida=0x585cdb
func TestSheepRanchClothesGolden(t *testing.T) {
	input := NewSheepRanchClothes(0x000003E8, 0x01)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0xE8, 0x03, 0x00, 0x00, 0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSheepRanchClothesRoundTrip(t *testing.T) {
	input := NewSheepRanchClothes(0x000003E8, 0x01)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
