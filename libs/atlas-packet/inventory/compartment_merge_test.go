package inventory

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCompartmentMergeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CompartmentMerge{updateTime: 100, compartmentType: 1}
			output := CompartmentMerge{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CompartmentType() != input.CompartmentType() {
				t.Errorf("compartmentType: got %v, want %v", output.CompartmentType(), input.CompartmentType())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
