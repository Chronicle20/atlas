package inventory

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestScrollUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ScrollUse{updateTime: 12345, scrollSlot: 3, equipSlot: -5, bWhiteScroll: 2, legendarySpirit: true}
			output := ScrollUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.ScrollSlot() != input.ScrollSlot() {
				t.Errorf("scrollSlot: got %v, want %v", output.ScrollSlot(), input.ScrollSlot())
			}
			if output.EquipSlot() != input.EquipSlot() {
				t.Errorf("equipSlot: got %v, want %v", output.EquipSlot(), input.EquipSlot())
			}
			if output.WhiteScroll() != input.WhiteScroll() {
				t.Errorf("whiteScroll: got %v, want %v", output.WhiteScroll(), input.WhiteScroll())
			}
			if output.LegendarySpirit() != input.LegendarySpirit() {
				t.Errorf("legendarySpirit: got %v, want %v", output.LegendarySpirit(), input.LegendarySpirit())
			}
		})
	}
}
