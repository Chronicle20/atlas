package trade

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestCommandEnvelopeJsonShape pins the wire contract atlas-channel mirrors.
// A field-name or tag drift here silently breaks the channel's decode into a
// zero-valued body.
func TestCommandEnvelopeJsonShape(t *testing.T) {
	c := Command[PutItemCommandBody]{
		TransactionId: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		WorldId:       1,
		ChannelId:     2,
		MapId:         100000000,
		CharacterId:   100,
		Type:          CommandTypePutItem,
		Body:          PutItemCommandBody{InventoryType: 2, Slot: 1, Quantity: 5, TargetSlot: 3},
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]interface{}
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"transactionId", "worldId", "channelId", "mapId", "instance", "characterId", "type", "body"} {
		if _, ok := round[k]; !ok {
			t.Errorf("command envelope missing key %q", k)
		}
	}
	body := round["body"].(map[string]interface{})
	for _, k := range []string{"inventoryType", "slot", "quantity", "targetSlot"} {
		if _, ok := body[k]; !ok {
			t.Errorf("put-item body missing key %q", k)
		}
	}
}

// TestStatusEventEnvelopeJsonShape pins the atlas-trades -> atlas-channel
// envelope. The channel addresses both participants straight off this envelope,
// so a dropped ownerId/visitorId tag would silently address nobody.
func TestStatusEventEnvelopeJsonShape(t *testing.T) {
	e := StatusEvent[ItemStagedEventBody]{
		TransactionId: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		WorldId:       1,
		ChannelId:     2,
		MapId:         100000000,
		RoomId:        uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Handle:        7,
		RoomType:      3,
		OwnerId:       100,
		VisitorId:     200,
		CharacterId:   100,
		Type:          StatusTypeItemStaged,
		Body: ItemStagedEventBody{
			Position: 0, TradeSlot: 1, InventoryType: 2, SourceSlot: 4,
			AssetId: 55, TemplateId: 2000000, Quantity: 5,
		},
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]interface{}
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"transactionId", "worldId", "channelId", "mapId", "instance", "roomId", "handle", "roomType", "ownerId", "visitorId", "characterId", "type", "body"} {
		if _, ok := round[k]; !ok {
			t.Errorf("status envelope missing key %q", k)
		}
	}
	body := round["body"].(map[string]interface{})
	for _, k := range []string{"position", "tradeSlot", "inventoryType", "sourceSlot", "assetId", "templateId", "quantity"} {
		if _, ok := body[k]; !ok {
			t.Errorf("item-staged body missing key %q", k)
		}
	}
}

// TestEveryCommandTypeIsDistinct guards against a copy-paste collision in the
// const block — two commands sharing a type string means one handler silently
// swallows the other's messages.
func TestEveryCommandTypeIsDistinct(t *testing.T) {
	all := []string{
		CommandTypeCreateRoom, CommandTypeInvite, CommandTypeDeclineInvite,
		CommandTypeEnterRoom, CommandTypePutItem, CommandTypeAddMeso,
		CommandTypeConfirm, CommandTypeTransaction, CommandTypeCancel, CommandTypeChat,
	}
	seen := make(map[string]bool, len(all))
	for _, v := range all {
		if v == "" {
			t.Fatal("empty command type constant")
		}
		if seen[v] {
			t.Errorf("duplicate command type %q", v)
		}
		seen[v] = true
	}
}

// TestEveryStatusTypeIsDistinct is the mirror image over the status types: a
// collision there means one status event is routed to the wrong channel-side
// handler, or silently swallowed by it.
func TestEveryStatusTypeIsDistinct(t *testing.T) {
	all := []string{
		StatusTypeRoomCreated, StatusTypeInviteSent, StatusTypeInviteRejected,
		StatusTypeParticipantEntered, StatusTypeItemStaged, StatusTypeMesoStaged,
		StatusTypeMesoRefused, StatusTypeParticipantConfirmed,
		StatusTypeAttestationRequested, StatusTypeSettled, StatusTypeCancelled,
		StatusTypeError, StatusTypeChat,
	}
	seen := make(map[string]bool, len(all))
	for _, v := range all {
		if v == "" {
			t.Fatal("empty status type constant")
		}
		if seen[v] {
			t.Errorf("duplicate status type %q", v)
		}
		seen[v] = true
	}
}
