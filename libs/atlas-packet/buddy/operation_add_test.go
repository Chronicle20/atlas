package buddy

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationAddRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationAdd{name: "TestBuddy", group: "Default Group"}
			output := OperationAdd{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Group() != input.Group() {
				t.Errorf("group: got %v, want %v", output.Group(), input.Group())
			}
		})
	}
}
