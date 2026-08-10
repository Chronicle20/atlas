package trade

import (
	trade2 "atlas-channel/kafka/message/trade"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const (
	testCharacterId = character.Id(1000)
	testTargetId    = character.Id(2000)
)

var testInstance = uuid.MustParse("6f1b0a1a-0000-4000-8000-000000000001")

func testField() field.Model {
	return field.NewBuilder(1, 2, 100000000).SetInstance(testInstance).Build()
}

// decodeOne runs a provider and unmarshals its single message back into the
// command envelope. Marshalling is the real boundary: atlas-trades sees the
// json, not the struct.
func decodeOne[E any](t *testing.T, p model.Provider[[]kafka.Message]) (trade2.Command[E], kafka.Message) {
	t.Helper()
	ms, err := p()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(ms) != 1 {
		t.Fatalf("messages: got %d, want 1", len(ms))
	}
	var c trade2.Command[E]
	if err := json.Unmarshal(ms[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return c, ms[0]
}

// TestCreateRoomCommandCarriesTheEnvelopeAndRoomType pins the envelope every
// other command shares: the acting character, the field it acted in, and the
// discriminating Type that atlas-trades' shared-topic handlers filter on.
func TestCreateRoomCommandCarriesTheEnvelopeAndRoomType(t *testing.T) {
	txId := uuid.New()
	c, _ := decodeOne[trade2.CreateRoomCommandBody](t, CreateRoomCommandProvider(txId, testField(), testCharacterId, 3))
	if c.Type != trade2.CommandTypeCreateRoom {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeCreateRoom)
	}
	if c.TransactionId != txId {
		t.Errorf("transactionId: got %s, want %s", c.TransactionId, txId)
	}
	if c.CharacterId != testCharacterId {
		t.Errorf("characterId: got %d, want %d", c.CharacterId, testCharacterId)
	}
	if c.WorldId != 1 || c.ChannelId != 2 || c.MapId != 100000000 || c.Instance != testInstance {
		t.Errorf("field: got world [%d] channel [%d] map [%d] instance [%s]", c.WorldId, c.ChannelId, c.MapId, c.Instance)
	}
	if c.Body.RoomType != 3 {
		t.Errorf("roomType: got %d, want 3", c.Body.RoomType)
	}
}

// TestCreateRoomCommandCarriesTheCashRoomType pins that the cash-trade room
// type reaches atlas-trades unchanged — the two room kinds share every command.
func TestCreateRoomCommandCarriesTheCashRoomType(t *testing.T) {
	c, _ := decodeOne[trade2.CreateRoomCommandBody](t, CreateRoomCommandProvider(uuid.New(), testField(), testCharacterId, 6))
	if c.Body.RoomType != 6 {
		t.Errorf("roomType: got %d, want 6", c.Body.RoomType)
	}
}

// TestEveryCommandIsKeyedByTheActingCharacter pins the ordering guarantee the
// cash-trade open depends on: CREATE_ROOM and INVITE are sent back-to-back and
// must land on the same partition, in order.
func TestEveryCommandIsKeyedByTheActingCharacter(t *testing.T) {
	want := producer.CreateKey(int(testCharacterId))
	providers := map[string]model.Provider[[]kafka.Message]{
		"createRoom":    CreateRoomCommandProvider(uuid.New(), testField(), testCharacterId, 3),
		"invite":        InviteCommandProvider(uuid.New(), testField(), testCharacterId, testTargetId),
		"declineInvite": DeclineInviteCommandProvider(uuid.New(), testField(), testCharacterId, 7, 1),
		"enterRoom":     EnterRoomCommandProvider(uuid.New(), testField(), testCharacterId, 7),
		"putItem":       PutItemCommandProvider(uuid.New(), testField(), testCharacterId, inventory.TypeValueUse, 5, 100, 3),
		"addMeso":       AddMesoCommandProvider(uuid.New(), testField(), testCharacterId, 10),
		"confirm":       ConfirmCommandProvider(uuid.New(), testField(), testCharacterId, nil),
		"transaction":   TransactionCommandProvider(uuid.New(), testField(), testCharacterId, nil),
		"cancel":        CancelCommandProvider(uuid.New(), testField(), testCharacterId),
		"chat":          ChatCommandProvider(uuid.New(), testField(), testCharacterId, "hi"),
	}
	for name, p := range providers {
		ms, err := p()
		if err != nil {
			t.Fatalf("%s: provider: %v", name, err)
		}
		if string(ms[0].Key) != string(want) {
			t.Errorf("%s: key: got %v, want %v", name, ms[0].Key, want)
		}
	}
}

// TestInviteCommandCarriesTheTarget pins the invitee id, which is the only
// field distinguishing an invite from a create.
func TestInviteCommandCarriesTheTarget(t *testing.T) {
	c, _ := decodeOne[trade2.InviteCommandBody](t, InviteCommandProvider(uuid.New(), testField(), testCharacterId, testTargetId))
	if c.Type != trade2.CommandTypeInvite {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeInvite)
	}
	if c.Body.TargetCharacterId != testTargetId {
		t.Errorf("targetCharacterId: got %d, want %d", c.Body.TargetCharacterId, testTargetId)
	}
}

// TestDeclineInviteCommandCarriesTheSerialAndErrorCode pins both decoded
// fields: atlas-trades resolves the room from the serial, and dropping the
// error code would lose the reason the client gave.
func TestDeclineInviteCommandCarriesTheSerialAndErrorCode(t *testing.T) {
	c, _ := decodeOne[trade2.DeclineInviteCommandBody](t, DeclineInviteCommandProvider(uuid.New(), testField(), testCharacterId, 4242, 3))
	if c.Type != trade2.CommandTypeDeclineInvite {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeDeclineInvite)
	}
	if c.Body.SerialNumber != 4242 {
		t.Errorf("serialNumber: got %d, want 4242", c.Body.SerialNumber)
	}
	if c.Body.ErrorCode != 3 {
		t.Errorf("errorCode: got %d, want 3", c.Body.ErrorCode)
	}
}

// TestEnterRoomCommandCarriesTheHandle pins the wire handle, which is what
// atlas-trades looks the room up by.
func TestEnterRoomCommandCarriesTheHandle(t *testing.T) {
	c, _ := decodeOne[trade2.EnterRoomCommandBody](t, EnterRoomCommandProvider(uuid.New(), testField(), testCharacterId, 9999))
	if c.Type != trade2.CommandTypeEnterRoom {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeEnterRoom)
	}
	if c.Body.Handle != 9999 {
		t.Errorf("handle: got %d, want 9999", c.Body.Handle)
	}
}

// TestPutItemCommandForwardsEveryDecodedField pins that no field is dropped
// between the codec and the command.
func TestPutItemCommandForwardsEveryDecodedField(t *testing.T) {
	c, _ := decodeOne[trade2.PutItemCommandBody](t, PutItemCommandProvider(uuid.New(), testField(), testCharacterId, inventory.TypeValueUse, 5, 100, 3))
	if c.Type != trade2.CommandTypePutItem {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypePutItem)
	}
	if c.Body.InventoryType != inventory.TypeValueUse {
		t.Errorf("inventoryType: got %d, want %d", c.Body.InventoryType, inventory.TypeValueUse)
	}
	if c.Body.Slot != 5 {
		t.Errorf("slot: got %d, want 5", c.Body.Slot)
	}
	if c.Body.Quantity != 100 {
		t.Errorf("quantity: got %d, want 100", c.Body.Quantity)
	}
	if c.Body.TargetSlot != 3 {
		t.Errorf("targetSlot: got %d, want 3", c.Body.TargetSlot)
	}
}

// TestPutItemCommandKeepsAnEquipSlotNegative pins that an equipped source slot
// survives the json round-trip: slot.Position is signed and equipment lives at
// negative positions.
func TestPutItemCommandKeepsAnEquipSlotNegative(t *testing.T) {
	c, _ := decodeOne[trade2.PutItemCommandBody](t, PutItemCommandProvider(uuid.New(), testField(), testCharacterId, inventory.TypeValueEquip, -11, 1, 1))
	if c.Body.Slot != -11 {
		t.Errorf("slot: got %d, want -11", c.Body.Slot)
	}
}

// TestAddMesoCommandKeepsANegativeAmount pins that a hostile client's negative
// total is forwarded as sent rather than clamped here — atlas-trades owns the
// rejection (kafka.go's AddMesoCommandBody doc).
func TestAddMesoCommandKeepsANegativeAmount(t *testing.T) {
	c, _ := decodeOne[trade2.AddMesoCommandBody](t, AddMesoCommandProvider(uuid.New(), testField(), testCharacterId, -1))
	if c.Type != trade2.CommandTypeAddMeso {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeAddMeso)
	}
	if c.Body.Amount != -1 {
		t.Errorf("amount: got %d, want -1", c.Body.Amount)
	}
}

// TestConfirmCommandForwardsTheCrcEntries pins that the attestation payload
// survives in order and in full.
func TestConfirmCommandForwardsTheCrcEntries(t *testing.T) {
	entries := []trade2.CrcEntry{{Data: 100, Crc: 200}, {Data: 300, Crc: 400}}
	c, _ := decodeOne[trade2.ConfirmCommandBody](t, ConfirmCommandProvider(uuid.New(), testField(), testCharacterId, entries))
	if c.Type != trade2.CommandTypeConfirm {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeConfirm)
	}
	if len(c.Body.Entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(c.Body.Entries))
	}
	if c.Body.Entries[0] != entries[0] || c.Body.Entries[1] != entries[1] {
		t.Errorf("entries: got %+v, want %+v", c.Body.Entries, entries)
	}
}

// TestConfirmCommandSurvivesAnEmptyCrcList pins the GMS <= v79 shape: those
// versions' TRADE_CONFIRM carries no CRC block (tradeCrcPresent), so an empty
// list is a faithful forward rather than a dropped field.
func TestConfirmCommandSurvivesAnEmptyCrcList(t *testing.T) {
	c, _ := decodeOne[trade2.ConfirmCommandBody](t, ConfirmCommandProvider(uuid.New(), testField(), testCharacterId, []trade2.CrcEntry{}))
	if c.Type != trade2.CommandTypeConfirm {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeConfirm)
	}
	if len(c.Body.Entries) != 0 {
		t.Errorf("entries: got %d, want 0", len(c.Body.Entries))
	}
}

// TestTransactionCommandForwardsTheCrcEntries pins the attestation reply, which
// is a distinct command type from CONFIRM even though the payload matches.
func TestTransactionCommandForwardsTheCrcEntries(t *testing.T) {
	entries := []trade2.CrcEntry{{Data: 7, Crc: 8}}
	c, _ := decodeOne[trade2.TransactionCommandBody](t, TransactionCommandProvider(uuid.New(), testField(), testCharacterId, entries))
	if c.Type != trade2.CommandTypeTransaction {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeTransaction)
	}
	if len(c.Body.Entries) != 1 || c.Body.Entries[0] != entries[0] {
		t.Errorf("entries: got %+v, want %+v", c.Body.Entries, entries)
	}
}

// TestCancelCommandIsTypedAsCancel pins the EXIT arm's command: the body is
// empty, so the type is the entire signal.
func TestCancelCommandIsTypedAsCancel(t *testing.T) {
	c, _ := decodeOne[trade2.CancelCommandBody](t, CancelCommandProvider(uuid.New(), testField(), testCharacterId))
	if c.Type != trade2.CommandTypeCancel {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeCancel)
	}
	if c.CharacterId != testCharacterId {
		t.Errorf("characterId: got %d, want %d", c.CharacterId, testCharacterId)
	}
}

// TestChatCommandCarriesTheMessage pins the chat fan-out's payload.
func TestChatCommandCarriesTheMessage(t *testing.T) {
	c, _ := decodeOne[trade2.ChatCommandBody](t, ChatCommandProvider(uuid.New(), testField(), testCharacterId, "hello"))
	if c.Type != trade2.CommandTypeChat {
		t.Errorf("type: got %s, want %s", c.Type, trade2.CommandTypeChat)
	}
	if c.Body.Message != "hello" {
		t.Errorf("message: got %q, want %q", c.Body.Message, "hello")
	}
}
