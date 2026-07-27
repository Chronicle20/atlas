package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body (slot int16, itemId int32;
// no updateTime). The body is identical on every version that carries the
// dedicated opcode, so no version gating is needed — Encode2(nPos)+Encode4(nItemID)
// under a CanSendExclRequest(200,0) guard. The opcode was introduced at v72;
// v48/v61 lack it and route reward boxes through the generic item-use request
// instead (see atlas-consumables RequestItemConsume).
//
// Client read order IDA-verified per version (task-131). Each send is a distinct
// 0x83-byte function, confirmed separate from the same-signature bridle send:
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=gms_v72 ida=0x90c93a
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=gms_v79 ida=0x95dd02
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=gms_v83 ida=0xa1249f
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=gms_v84 ida=0xa5c8dc
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=gms_v87 ida=0xaa7ec6
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=gms_v95 ida=0x9d6c50
// packet-audit:verify packet=inventory/serverbound/InventoryLotteryItemUse version=jms_v185 ida=0xaf6900
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
