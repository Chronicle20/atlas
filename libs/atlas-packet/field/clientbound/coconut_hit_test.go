package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldCoconutHit version=gms_v83 ida=0x549834
// packet-audit:verify packet=field/clientbound/FieldCoconutHit version=gms_v84 ida=0x555fa7
// packet-audit:verify packet=field/clientbound/FieldCoconutHit version=gms_v87 ida=0x5734e9
// packet-audit:verify packet=field/clientbound/FieldCoconutHit version=gms_v95 ida=0x54a470
// packet-audit:verify packet=field/clientbound/FieldCoconutHit version=jms_v185 ida=0x589b03
func TestCoconutHitGolden(t *testing.T) {
	input := NewCoconutHit(0x0005, 0x0001, 0x03)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x05, 0x00, 0x01, 0x00, 0x03}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestCoconutHitRoundTrip(t *testing.T) {
	input := NewCoconutHit(0x0005, 0x0001, 0x03)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
