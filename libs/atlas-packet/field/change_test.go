package field

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestChangeWithPortalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{fieldKey: 1, targetId: 100000000, portalName: "west00", unused: 0, premium: 0}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FieldKey() != input.FieldKey() {
				t.Errorf("fieldKey: got %v, want %v", output.FieldKey(), input.FieldKey())
			}
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.PortalName() != input.PortalName() {
				t.Errorf("portalName: got %v, want %v", output.PortalName(), input.PortalName())
			}
		})
	}
}

func TestChangeWithPositionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{fieldKey: 1, targetId: 100000000, portalName: "", x: 100, y: 200, unused: 0, premium: 0}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
