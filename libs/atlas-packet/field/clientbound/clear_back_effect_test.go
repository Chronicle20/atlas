package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v61 ida=0x5a871b
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v72 ida=0x5f5f54
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v79 ida=0x614977
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v83 ida=0x6449ca
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v84 ida=0x65a241
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v87 ida=0x67e0e0
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v92 ida=0x612ef0
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=gms_v95 ida=0x61f230
// packet-audit:verify packet=field/clientbound/FieldClearBackEffect version=jms_v185 ida=0x6ba684
func TestClearBackEffectGolden(t *testing.T) {
	input := NewClearBackEffect()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestClearBackEffectRoundTrip(t *testing.T) {
	input := NewClearBackEffect()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
