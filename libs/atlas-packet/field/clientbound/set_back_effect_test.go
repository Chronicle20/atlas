package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v61 ida=0x5a8316
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v72 ida=0x5f5b4f
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v79 ida=0x614572
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v83 ida=0x6445c5
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v84 ida=0x659e3c
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v87 ida=0x67dcdb
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v92 ida=0x606d80
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=gms_v95 ida=0x612850
// packet-audit:verify packet=field/clientbound/FieldSetBackEffect version=jms_v185 ida=0x6ba27f
func TestSetBackEffectGolden(t *testing.T) {
	input := NewSetBackEffect(BackEffectShow, 100000000, 1, 1000)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x00, 0x00, 0xE1, 0xF5, 0x05, 0x01, 0xE8, 0x03, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSetBackEffectRoundTrip(t *testing.T) {
	input := NewSetBackEffect(BackEffectShow, 100000000, 1, 1000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
