package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body (slot int16, itemId int32;
// no updateTime). Client read order IDA-verified during design: v83 fn 0xa1249f,
// v95 fn 0x9d6c50 (design task-131 §2.1). The packet-audit:verify markers +
// matrix-cell promotion are deferred to a follow-up: CWvsContext::SendLotteryItemUseRequest
// is not present in the checked-in IDA exports, so evidence records cannot be
// pinned without a live-IDA re-export (task-131 plan Task 12, escalated).
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
