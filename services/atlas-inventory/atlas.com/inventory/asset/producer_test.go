package asset_test

import (
	"encoding/json"
	"testing"
	"time"

	"atlas-inventory/asset"
	assetmsg "atlas-inventory/kafka/message/asset"

	"github.com/google/uuid"
)

// TestMovedEventStatusProvider asserts that the provider materializes a
// kafka.Message whose decoded StatusEvent has Slot bound to the asset's NEW
// slot (its current slot post-update) and Body.OldSlot bound to the slot the
// asset was at before the update.
//
// This guards against the regression where the call site in UpdateSlot
// inadvertently passed (oldSlot, newSlot) into a producer declared as
// (newSlot, oldSlot, ...), inverting equip/unequip semantics for every
// downstream consumer that branches on IsEquipAction / IsUnequipAction
// (atlas-effective-stats, atlas-rates).
func TestMovedEventStatusProvider(t *testing.T) {
	cases := []struct {
		name    string
		oldSlot int16
		newSlot int16
	}{
		// Unequip: equip slot (negative) -> inventory slot (positive). This is
		// the production trace shape ("Could not find equipment data for
		// asset [177]") that surfaced the bug.
		{name: "unequip", oldSlot: -17, newSlot: 3},
		// Equip: inventory slot (positive) -> equip slot (negative).
		{name: "equip", oldSlot: 3, newSlot: -17},
		// Inventory rearrange: both positive. Not slot-sensitive downstream
		// but locks in correctness regardless of sign.
		{name: "rearrange", oldSlot: 5, newSlot: 12},
	}

	transactionId := uuid.New()
	compartmentId := uuid.New()
	const characterId = uint32(1000042)
	const assetId = uint32(177)
	const templateId = uint32(1040010)
	createdAt := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			provider := asset.MovedEventStatusProvider(
				transactionId,
				characterId,
				compartmentId,
				assetId,
				templateId,
				tc.newSlot,
				tc.oldSlot,
				createdAt,
			)

			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider returned error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}

			var event assetmsg.StatusEvent[assetmsg.MovedStatusEventBody]
			if err := json.Unmarshal(msgs[0].Value, &event); err != nil {
				t.Fatalf("failed to decode message value: %v", err)
			}

			if event.Type != assetmsg.StatusEventTypeMoved {
				t.Errorf("Type: got %q, want %q", event.Type, assetmsg.StatusEventTypeMoved)
			}
			if event.AssetId != assetId {
				t.Errorf("AssetId: got %d, want %d", event.AssetId, assetId)
			}
			if event.TemplateId != templateId {
				t.Errorf("TemplateId: got %d, want %d", event.TemplateId, templateId)
			}
			if event.CharacterId != characterId {
				t.Errorf("CharacterId: got %d, want %d", event.CharacterId, characterId)
			}
			if event.Slot != tc.newSlot {
				t.Errorf("Slot (NEW): got %d, want %d", event.Slot, tc.newSlot)
			}
			if event.Body.OldSlot != tc.oldSlot {
				t.Errorf("Body.OldSlot (OLD): got %d, want %d", event.Body.OldSlot, tc.oldSlot)
			}
		})
	}
}
