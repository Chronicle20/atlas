package consumable

import (
	"encoding/json"
	"testing"

	consumablemsg "atlas-channel/kafka/message/consumable"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestRequestItemConsumeCommandProvider_CarriesQuantity pins that the
// REQUEST_ITEM_CONSUME command body carries the caller's quantity on the
// wire — the skill-cast itemCon path now sends values > 1 (FR-1).
func TestRequestItemConsumeCommandProvider_CarriesQuantity(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	cid := character.Id(42)

	provider := RequestItemConsumeCommandProvider(f, cid, slot.Position(7), item.Id(4006000), int16(2))
	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd consumablemsg.Command[consumablemsg.RequestItemConsumeBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Body.Source != slot.Position(7) {
		t.Errorf("source: got %d, want 7", cmd.Body.Source)
	}
	if cmd.Body.ItemId != item.Id(4006000) {
		t.Errorf("itemId: got %d, want 4006000", cmd.Body.ItemId)
	}
	if cmd.Body.Quantity != 2 {
		t.Errorf("quantity: got %d, want 2", cmd.Body.Quantity)
	}
}
