package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldMtsOperation2 version=gms_v83 ida=0x5a428c
// packet-audit:verify packet=field/clientbound/FieldMtsOperation2 version=gms_v84 ida=0x5b4743
// packet-audit:verify packet=field/clientbound/FieldMtsOperation2 version=gms_v87 ida=0x5d434b
// packet-audit:verify packet=field/clientbound/FieldMtsOperation2 version=gms_v95 ida=0x575c40
func TestMtsOperation2Golden(t *testing.T) {
	input := NewMtsOperation2(0x01020304, 0x05060708)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMtsOperation2RoundTrip(t *testing.T) {
	input := NewMtsOperation2(0x01020304, 0x05060708)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
