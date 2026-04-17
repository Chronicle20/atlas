package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSetGenderRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/set_true", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetGender{set: true, gender: 1}
			output := SetGender{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Set() != input.Set() {
				t.Errorf("set: got %v, want %v", output.Set(), input.Set())
			}
			if output.Gender() != input.Gender() {
				t.Errorf("gender: got %v, want %v", output.Gender(), input.Gender())
			}
		})
		t.Run(v.Name+"/set_false", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetGender{set: false}
			output := SetGender{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Set() != input.Set() {
				t.Errorf("set: got %v, want %v", output.Set(), input.Set())
			}
		})
	}
}
