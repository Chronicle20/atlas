package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterSitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterSit(100)
			output := CharacterSitResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.Sitting() {
				t.Errorf("sitting: got false, want true")
			}
			if output.ChairId() != input.ChairId() {
				t.Errorf("chairId: got %v, want %v", output.ChairId(), input.ChairId())
			}
		})
	}
}

func TestCharacterCancelSitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterCancelSit()
			output := CharacterSitResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Sitting() {
				t.Errorf("sitting: got true, want false")
			}
		})
	}
}
