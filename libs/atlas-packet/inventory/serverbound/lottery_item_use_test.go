package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body (slot int16, itemId int32;
// no updateTime). The body is identical on every version that carries the
// dedicated opcode, so no version gating is needed. Client read order is
// IDA-verified across the version set: v83 fn 0xa1249f, v95 fn 0x9d6c50 (design
// task-131 §2.1); v72 fn 0x90c93a (opcode 0x6F), v79 fn 0x95dd02 (opcode 0x6E),
// jms fn 0xaf6900 (opcode 0x6B) — all Encode2(nPos)+Encode4(nItemID), verified
// live this session (task-131 main-merge scope expansion). The opcode was
// introduced at v72; v48/v61 lack it and route reward boxes through the generic
// item-use request instead (see atlas-consumables RequestItemConsume).
//
// The verify markers and ✅ matrix promotion remain a uniform follow-up for ALL
// cells: CWvsContext::SendLotteryItemUseRequest is absent from the checked-in IDA
// exports, so evidence records cannot be pinned without a live-IDA re-export
// (task-131 plan Task 12, escalated). Cells are "incomplete" (routed + codec)
// until that export refresh.
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
