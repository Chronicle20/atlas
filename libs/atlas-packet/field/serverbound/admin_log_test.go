package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldAdminLog version=gms_v83 ida=0x52e297
// packet-audit:verify packet=field/serverbound/FieldAdminLog version=gms_v84 ida=0x53a38e
// packet-audit:verify packet=field/serverbound/FieldAdminLog version=gms_v87 ida=0x55524f
// packet-audit:verify packet=field/serverbound/FieldAdminLog version=gms_v95 ida=0x54298b
// packet-audit:verify packet=field/serverbound/FieldAdminLog version=jms_v185 ida=0x56a838
func TestAdminLogGolden(t *testing.T) {
	input := NewAdminLog("hi")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAdminLogRoundTrip(t *testing.T) {
	input := NewAdminLog("hi")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := AdminLog{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
