package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldViciousHammer version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammer version=gms_v84 ida=0x544395
// packet-audit:verify packet=field/clientbound/FieldViciousHammer version=gms_v87 ida=0x55fa12
// packet-audit:verify packet=field/clientbound/FieldViciousHammer version=gms_v95 ida=0x52a430
func TestViciousHammerGolden(t *testing.T) {
	input := NewViciousHammer()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestViciousHammerRoundTrip(t *testing.T) {
	input := NewViciousHammer()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
