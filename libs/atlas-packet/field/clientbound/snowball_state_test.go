package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v83 ida=0x5750a3
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v84 ida=0x584a1c
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v87 ida=0x5a3328
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v95 ida=0x560ab0
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=jms_v185 ida=0x5c959d
func TestSnowballStateGolden(t *testing.T) {
	input := NewSnowballState(0x01, 0x00000064, 0x00000032, 0x0190, 0x02, 0x0003, 0x0004, 0x0005)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x64, 0x00, 0x00, 0x00, 0x32, 0x00, 0x00, 0x00, 0x90, 0x01, 0x02, 0x03, 0x00, 0x04, 0x00, 0x05, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSnowballStateRoundTrip(t *testing.T) {
	input := NewSnowballState(0x01, 0x00000064, 0x00000032, 0x0190, 0x02, 0x0003, 0x0004, 0x0005)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
