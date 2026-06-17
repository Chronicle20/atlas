package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v83 ida=0x52ec6c
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v84 ida=0x53ad6d
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v87 ida=0x555dff
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v95 ida=0x5445eb
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=jms_v185 ida=0x56b971
func TestMatchTableGolden(t *testing.T) {
	input := NewMatchTable(0x01)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMatchTableRoundTrip(t *testing.T) {
	input := NewMatchTable(0x01)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MatchTable{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Flag() != input.Flag() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
