package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterHintRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterHint("Hello World", 200, 50, false, 0, 0)
			output := CharacterHint{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Hint() != input.Hint() {
				t.Errorf("hint: got %v, want %v", output.Hint(), input.Hint())
			}
			if output.Width() != input.Width() {
				t.Errorf("width: got %v, want %v", output.Width(), input.Width())
			}
		})
	}
}

func TestCharacterHintAtPointRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterHint("At Point", 200, 50, true, 100, 200)
			output := CharacterHint{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.AtPoint() {
				t.Errorf("atPoint: got false, want true")
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
