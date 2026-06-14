package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSnowballHit version=gms_v83 ida=0x575191
// packet-audit:verify packet=field/clientbound/FieldSnowballHit version=gms_v84 ida=0x584b0a
// packet-audit:verify packet=field/clientbound/FieldSnowballHit version=gms_v87 ida=0x5a3416
// packet-audit:verify packet=field/clientbound/FieldSnowballHit version=gms_v95 ida=0x5619d0
// packet-audit:verify packet=field/clientbound/FieldSnowballHit version=jms_v185 ida=0x5c968b
func TestSnowballHitGolden(t *testing.T) {
	input := NewSnowballHit(0x01, 0x000A, 0x0014)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x0A, 0x00, 0x14, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSnowballHitRoundTrip(t *testing.T) {
	input := NewSnowballHit(0x01, 0x000A, 0x0014)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
