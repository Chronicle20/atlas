package trade

import (
	trade2 "atlas-channel/kafka/message/trade"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// commandProvider builds one COMMAND_TOPIC_TRADE message. Every command is
// keyed by the ACTING character, so all of one character's trade commands land
// on the same partition and atlas-trades observes them in send order — which is
// what makes CREATE_ROOM followed immediately by INVITE (the cash-trade open)
// safe.
func commandProvider[E any](transactionId uuid.UUID, f field.Model, characterId character.Id, commandType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &trade2.Command[E]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          commandType,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}

func CreateRoomCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, roomType byte) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeCreateRoom, trade2.CreateRoomCommandBody{
		RoomType: roomType,
	})
}

func InviteCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, targetCharacterId character.Id) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeInvite, trade2.InviteCommandBody{
		TargetCharacterId: targetCharacterId,
	})
}

func DeclineInviteCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, serialNumber uint32, errorCode byte) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeDeclineInvite, trade2.DeclineInviteCommandBody{
		SerialNumber: serialNumber,
		ErrorCode:    errorCode,
	})
}

func EnterRoomCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, handle uint32) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeEnterRoom, trade2.EnterRoomCommandBody{
		Handle: handle,
	})
}

func PutItemCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, quantity uint16, targetSlot byte) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypePutItem, trade2.PutItemCommandBody{
		InventoryType: inventoryType,
		Slot:          sourceSlot,
		Quantity:      quantity,
		TargetSlot:    targetSlot,
	})
}

func AddMesoCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, amount int32) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeAddMeso, trade2.AddMesoCommandBody{
		Amount: amount,
	})
}

func ConfirmCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, entries []trade2.CrcEntry) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeConfirm, trade2.ConfirmCommandBody{
		Entries: entries,
	})
}

func TransactionCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, entries []trade2.CrcEntry) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeTransaction, trade2.TransactionCommandBody{
		Entries: entries,
	})
}

func CancelCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeCancel, trade2.CancelCommandBody{})
}

func ChatCommandProvider(transactionId uuid.UUID, f field.Model, characterId character.Id, message string) model.Provider[[]kafka.Message] {
	return commandProvider(transactionId, f, characterId, trade2.CommandTypeChat, trade2.ChatCommandBody{
		Message: message,
	})
}
