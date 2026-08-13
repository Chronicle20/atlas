package consumable

import (
	"bytes"
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

// TestRequestItemConsumeWithPetCommandProvider_CarriesPetId pins the 0519 pet
// skill pouch path: the command body must carry the target petId, and must
// pin quantity 1 (the case-28 wire body has no quantity field, so the pouch
// is always a single consume — see libs/atlas-packet/cash/serverbound/
// item_use_pet_skill.go).
func TestRequestItemConsumeWithPetCommandProvider_CarriesPetId(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	cid := character.Id(42)

	provider := RequestItemConsumeWithPetCommandProvider(f, cid, slot.Position(3), item.Id(5190000), int16(1), uint64(987654321))
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
	if cmd.Type != consumablemsg.CommandRequestItemConsume {
		t.Errorf("type: got %q, want %q", cmd.Type, consumablemsg.CommandRequestItemConsume)
	}
	if cmd.Body.PetId != uint64(987654321) {
		t.Errorf("petId: got %d, want 987654321", cmd.Body.PetId)
	}
	if cmd.Body.ItemId != item.Id(5190000) {
		t.Errorf("itemId: got %d, want 5190000", cmd.Body.ItemId)
	}
	if cmd.Body.Quantity != 1 {
		t.Errorf("quantity: got %d, want 1", cmd.Body.Quantity)
	}
}

// TestRequestItemConsumeCommandProvider_OmitsPetId pins the backward-compatible
// half of the same contract: the non-pet consume path must not emit a petId
// key at all, so atlas-consumables' ConsumePetSkillPouch branch can never be
// entered by an ordinary consume. Asserted on the raw JSON because a zero
// uint64 and an absent key are indistinguishable after unmarshalling.
func TestRequestItemConsumeCommandProvider_OmitsPetId(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	provider := RequestItemConsumeCommandProvider(f, character.Id(42), slot.Position(7), item.Id(2000000), int16(1))
	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if bytes.Contains(msgs[0].Value, []byte("petId")) {
		t.Errorf("plain consume emitted a petId key: %s", msgs[0].Value)
	}
}
