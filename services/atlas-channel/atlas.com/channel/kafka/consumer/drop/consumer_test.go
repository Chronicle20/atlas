package drop

import (
	drop2 "atlas-channel/kafka/message/drop"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestIsConsumedOnPickupCard locks the classification that drives the
// monster-book-card pickup path: cards (item classification 238, e.g. 2380000)
// are consumed on pickup, so the handler suppresses the generic pickup message
// and sends an action-unlock; every other item keeps the normal pickup message.
func TestIsConsumedOnPickupCard(t *testing.T) {
	cases := []struct {
		name   string
		itemId uint32
		want   bool
	}{
		{name: "first monster-book card", itemId: 2380000, want: true},
		{name: "another monster-book card", itemId: 2380001, want: true},
		{name: "high monster-book card", itemId: 2389999, want: true},
		{name: "use-tab potion is not a card", itemId: 2000000, want: false},
		{name: "etc item is not a card", itemId: 4000000, want: false},
		{name: "equip is not a card", itemId: 1302000, want: false},
		{name: "zero (no item / meso pickup) is not a card", itemId: 0, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isConsumedOnPickupCard(tc.itemId); got != tc.want {
				t.Errorf("isConsumedOnPickupCard(%d) = %v, want %v", tc.itemId, got, tc.want)
			}
		})
	}
}

func nullLogger() *logrus.Logger {
	l, _ := testlog.NewNullLogger()
	return l
}

// TestPickupStatusMessagePacket_MesoOnly_NoAnnounce is a regression guard for
// the meso-notify fix: a meso-only pickup (ItemId/EquipmentId/Quantity all 0)
// must not produce a generic pickup CharacterStatusMessage, because
// MESO_AWARDED already wrote the DropPickUpMeso message with the correct
// per-recipient share (handleStatusEventMesoAwarded). Before this guard, the
// stackable-item branch encoded a spurious DropPickUpStackableItem(0, 0)
// packet for every meso-only pickup.
func TestPickupStatusMessagePacket_MesoOnly_NoAnnounce(t *testing.T) {
	body := drop2.PickedUpStatusEventBody{
		CharacterId: 6001,
		ItemId:      0,
		EquipmentId: 0,
		Quantity:    0,
		Meso:        471,
		PetSlot:     -1,
	}

	bp, ok := pickupStatusMessagePacket(body)
	if ok {
		t.Fatalf("meso-only pickup: got ok = true, want false (no CharacterStatusMessage should be announced)")
	}
	if bp != nil {
		t.Fatalf("meso-only pickup: got a non-nil packet.Encode, want nil")
	}
}

// TestPickupStatusMessagePacket_Item_Announces is the item-pickup companion
// to TestPickupStatusMessagePacket_MesoOnly_NoAnnounce: a normal (non-meso)
// item pickup must still produce a CharacterStatusMessage to announce.
func TestPickupStatusMessagePacket_Item_Announces(t *testing.T) {
	cases := []struct {
		name string
		body drop2.PickedUpStatusEventBody
	}{
		{
			name: "stackable item",
			body: drop2.PickedUpStatusEventBody{CharacterId: 6001, ItemId: 2000000, Quantity: 5, PetSlot: -1},
		},
		{
			name: "equipment (unstackable) item",
			body: drop2.PickedUpStatusEventBody{CharacterId: 6001, ItemId: 1302000, EquipmentId: 1000000123, PetSlot: -1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp, ok := pickupStatusMessagePacket(tc.body)
			if !ok {
				t.Fatalf("%s: got ok = false, want true (a CharacterStatusMessage should be announced)", tc.name)
			}
			if bp == nil {
				t.Fatalf("%s: got a nil packet.Encode, want non-nil", tc.name)
			}
			// The packet must actually encode without panicking.
			if got := bp(nullLogger(), context.Background())(map[string]interface{}{}); len(got) == 0 {
				t.Fatalf("%s: encoded packet is empty", tc.name)
			}
		})
	}
}
