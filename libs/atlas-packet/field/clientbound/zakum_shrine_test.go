package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldZakumShrine version=gms_v83 ida=0x53347c
// packet-audit:verify packet=field/clientbound/FieldZakumShrine version=gms_v84 ida=0x53f702
// packet-audit:verify packet=field/clientbound/FieldZakumShrine version=gms_v87 ida=0x55ac56
// packet-audit:verify packet=field/clientbound/FieldZakumShrine version=gms_v95 ida=0x530cc0
// packet-audit:verify packet=field/clientbound/FieldZakumShrine version=jms_v185 ida=0x57069f
func TestZakumShrineGolden(t *testing.T) {
	input := NewZakumShrine(0x01, 0x01020304)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x04, 0x03, 0x02, 0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestZakumShrineRoundTrip(t *testing.T) {
	input := NewZakumShrine(0x01, 0x01020304)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
