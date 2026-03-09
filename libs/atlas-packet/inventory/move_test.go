package inventory

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMoveRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Move{updateTime: 12345, inventoryType: 1, source: 5, destination: 10, count: 1}
			output := Move{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.Destination() != input.Destination() {
				t.Errorf("destination: got %v, want %v", output.Destination(), input.Destination())
			}
			if output.Count() != input.Count() {
				t.Errorf("count: got %v, want %v", output.Count(), input.Count())
			}
		})
	}
}
