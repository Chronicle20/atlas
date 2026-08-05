package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimAvailableTimeGolden(t *testing.T) {
	// openHour, closeHour. 0/0 = always available (verified client branch).
	input := NewClaimAvailableTime(8, 22)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x08, 0x16}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimAvailableTimeAlwaysOpenGolden(t *testing.T) {
	input := NewClaimAvailableTime(0, 0)
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x00, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimAvailableTimeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimAvailableTime(9, 21)
			output := ClaimAvailableTime{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OpenHour() != input.OpenHour() || output.CloseHour() != input.CloseHour() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
