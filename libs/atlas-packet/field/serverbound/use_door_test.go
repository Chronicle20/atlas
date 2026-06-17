package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v83 ida=0x5375ed
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v84 ida=0x5438eb
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v87 ida=0x55ef62
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v95 ida=0x52f970
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=jms_v185 ida=0x574826
func TestUseDoorGolden(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x05)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x05}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestUseDoorRoundTrip(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x05)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := UseDoor{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PortalFieldId() != input.PortalFieldId() || output.Flag() != input.Flag() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
