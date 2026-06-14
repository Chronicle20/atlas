package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldBlockedMap version=gms_v83 ida=0x53185c
// packet-audit:verify packet=field/clientbound/FieldBlockedMap version=gms_v84 ida=0x53dae2
// packet-audit:verify packet=field/clientbound/FieldBlockedMap version=gms_v87 ida=0x5590e1
// packet-audit:verify packet=field/clientbound/FieldBlockedMap version=gms_v95 ida=0x52f3b0
// packet-audit:verify packet=field/clientbound/FieldBlockedMap version=jms_v185 ida=0x56ec7b
func TestBlockedMapGolden(t *testing.T) {
	input := NewBlockedMap(0x07)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x07}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestBlockedMapRoundTrip(t *testing.T) {
	input := NewBlockedMap(0x07)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
