package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/LotteryItemUse version=gms_v83 ida=0xa1249f
// packet-audit:verify packet=inventory/serverbound/LotteryItemUse version=gms_v95 ida=0x9d6c50
func TestLotteryItemUseRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := LotteryItemUse{source: 5, itemId: 2022309}
			output := LotteryItemUse{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
