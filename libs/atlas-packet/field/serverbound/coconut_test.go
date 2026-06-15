package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v83 ida=0x549902
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v84 ida=0x556075
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v87 ida=0x5735b7
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v95 ida=0x54a5e0
// packet-audit:verify packet=field/serverbound/FieldCoconut version=jms_v185 ida=0x589bd1
func TestCoconutGolden(t *testing.T) {
	input := NewCoconut(0x0102, 0x0304)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02, 0x01, 0x04, 0x03}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestCoconutRoundTrip(t *testing.T) {
	input := NewCoconut(0x0102, 0x0304)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Coconut{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Attack() != input.Attack() || output.X() != input.X() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
