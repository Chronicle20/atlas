package serverbound

import (
	"bytes"
	"reflect"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldRequestFootholdInfo version=gms_v95 ida=0x52ddd0
// packet-audit:verify packet=field/serverbound/FieldRequestFootholdInfo version=jms_v185 ida=0x576cd2

// TestRequestFootholdInfoGolden encodes a single moving entry: nCurState +
// nCurX + nCurY + reverseVertical + reverseHorizontal (no count prefix).
func TestRequestFootholdInfoGolden(t *testing.T) {
	input := NewRequestFootholdInfo([]FootholdInfoEntry{
		NewFootholdInfoEntry(0x01020304, 0x05060708, 0x090A0B0C, 0x01, 0x00),
	})
	ctx := pt.CreateContext("GMS", 95, 0)
	expected := []byte{
		0x04, 0x03, 0x02, 0x01, // nCurState
		0x08, 0x07, 0x06, 0x05, // nCurX
		0x0C, 0x0B, 0x0A, 0x09, // nCurY
		0x01, // reverseVertical
		0x00, // reverseHorizontal
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch:\n got  %v\n want %v", actual, expected)
	}
}

func TestRequestFootholdInfoRoundTrip(t *testing.T) {
	input := NewRequestFootholdInfo([]FootholdInfoEntry{
		NewFootholdInfoEntry(1, 100, 200, 1, 0),
		NewFootholdInfoEntry(2, 0, 0, 0, 0),
	})
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := RequestFootholdInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !reflect.DeepEqual(output.Entries(), input.Entries()) {
				t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", output.Entries(), input.Entries())
			}
		})
	}
}
