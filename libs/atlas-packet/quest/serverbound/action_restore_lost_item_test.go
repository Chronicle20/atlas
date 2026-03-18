package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestActionRestoreLostItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionRestoreLostItem{unk1: 0, itemId: 4001000}
			output := ActionRestoreLostItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk1() != input.Unk1() {
				t.Errorf("unk1: got %v, want %v", output.Unk1(), input.Unk1())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
