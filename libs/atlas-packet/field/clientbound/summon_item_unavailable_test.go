package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSummonItemUnavailable version=gms_v83 ida=0x532fcf
// packet-audit:verify packet=field/clientbound/FieldSummonItemUnavailable version=gms_v84 ida=0x53f255
// packet-audit:verify packet=field/clientbound/FieldSummonItemUnavailable version=gms_v87 ida=0x55a7e8
// packet-audit:verify packet=field/clientbound/FieldSummonItemUnavailable version=gms_v95 ida=0x52f7b0
// packet-audit:verify packet=field/clientbound/FieldSummonItemUnavailable version=jms_v185 ida=0x570231
func TestSummonItemUnavailableGolden(t *testing.T) {
	input := NewSummonItemUnavailable(0x03)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x03}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSummonItemUnavailableRoundTrip(t *testing.T) {
	input := NewSummonItemUnavailable(0x03)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
