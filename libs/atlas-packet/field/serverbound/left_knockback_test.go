package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v83 ida=0x574df1
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v84 ida=0x58476f
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v87 ida=0x5a307b
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v95 ida=0x5612d0
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=jms_v185 ida=0x5c92fb
func TestLeftKnockbackGolden(t *testing.T) {
	input := NewLeftKnockback()
	ctx := pt.CreateContext("GMS", 83, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestLeftKnockbackRoundTrip(t *testing.T) {
	input := NewLeftKnockback()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := LeftKnockback{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
