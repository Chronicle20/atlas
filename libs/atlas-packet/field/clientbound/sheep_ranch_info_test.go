package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSheepRanchInfo version=gms_v83 ida=0x545c1e
// packet-audit:verify packet=field/clientbound/FieldSheepRanchInfo version=gms_v84 ida=0x5522cf
// packet-audit:verify packet=field/clientbound/FieldSheepRanchInfo version=gms_v87 ida=0x56f68e
// packet-audit:verify packet=field/clientbound/FieldSheepRanchInfo version=gms_v95 ida=0x5499a0
// packet-audit:verify packet=field/clientbound/FieldSheepRanchInfo version=jms_v185 ida=0x585ca7
func TestSheepRanchInfoGolden(t *testing.T) {
	input := NewSheepRanchInfo(0x03, 0x01)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x03, 0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSheepRanchInfoRoundTrip(t *testing.T) {
	input := NewSheepRanchInfo(0x03, 0x01)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
