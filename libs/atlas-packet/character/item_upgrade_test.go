package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestItemUpgradeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewItemUpgrade(12345, true, false, true, false)
			output := ItemUpgrade{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
			if output.Cursed() != input.Cursed() {
				t.Errorf("cursed: got %v, want %v", output.Cursed(), input.Cursed())
			}
			if output.LegendarySpirit() != input.LegendarySpirit() {
				t.Errorf("legendarySpirit: got %v, want %v", output.LegendarySpirit(), input.LegendarySpirit())
			}
			if output.WhiteScroll() != input.WhiteScroll() {
				t.Errorf("whiteScroll: got %v, want %v", output.WhiteScroll(), input.WhiteScroll())
			}
		})
	}
}
