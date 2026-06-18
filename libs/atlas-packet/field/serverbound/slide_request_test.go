package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldSlideRequest version=gms_v95 ida=0x542439
// packet-audit:verify packet=field/serverbound/FieldSlideRequest version=jms_v185 ida=0x5687f8
func TestSlideRequestGolden(t *testing.T) {
	input := NewSlideRequest(0x07)
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x07}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSlideRequestRoundTrip(t *testing.T) {
	input := NewSlideRequest(0x07)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := SlideRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Value() != input.Value() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
