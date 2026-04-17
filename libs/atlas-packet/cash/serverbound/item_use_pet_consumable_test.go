package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUsePetConsumableUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUsePetConsumable{updateTimeFirst: true}
			output := *NewItemUsePetConsumable(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestItemUsePetConsumableNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUsePetConsumable{updateTime: 12345, updateTimeFirst: false}
			output := *NewItemUsePetConsumable(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
