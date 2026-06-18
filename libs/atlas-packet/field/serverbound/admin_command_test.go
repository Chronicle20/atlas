package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v83 ida=0x52c958
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v84 ida=0x53891a
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v87 ida=0x5531b8
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v95 ida=0x540fbe
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=jms_v185 ida=0x568ac2
func TestAdminCommandGolden(t *testing.T) {
	input := NewAdminCommand(0x05)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x05}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAdminCommandRoundTrip(t *testing.T) {
	input := NewAdminCommand(0x05)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := AdminCommand{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SubCommand() != input.SubCommand() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
