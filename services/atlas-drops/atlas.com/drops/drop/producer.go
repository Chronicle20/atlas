package drop

import (
	messageDropKafka "atlas-drops/kafka/message/drop"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createdEventStatusProvider(drop Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(drop.Id()))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventCreatedBody]{
		TransactionId: drop.TransactionId(),
		WorldId:       drop.WorldId(),
		ChannelId:     drop.ChannelId(),
		MapId:         drop.MapId(),
		DropId:        drop.Id(),
		Type:          messageDropKafka.StatusEventTypeCreated,
		Body: messageDropKafka.StatusEventCreatedBody{
			ItemId:          drop.ItemId(),
			Quantity:        drop.Quantity(),
			Meso:            drop.Meso(),
			Type:            drop.Type(),
			X:               drop.X(),
			Y:               drop.Y(),
			OwnerId:         drop.OwnerId(),
			OwnerPartyId:    drop.OwnerPartyId(),
			DropTime:        drop.DropTime(),
			DropperUniqueId: drop.DropperId(),
			DropperX:        drop.DropperX(),
			DropperY:        drop.DropperY(),
			PlayerDrop:      drop.PlayerDrop(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func expiredEventStatusProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, dropId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventExpiredBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypeExpired,
		Body:          messageDropKafka.StatusEventExpiredBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func pickedUpEventStatusProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, dropId uint32, characterId uint32, itemId uint32, equipmentId uint32, quantity uint32, meso uint32, petSlot int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventPickedUpBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypePickedUp,
		Body: messageDropKafka.StatusEventPickedUpBody{
			CharacterId: characterId,
			ItemId:      itemId,
			EquipmentId: equipmentId,
			Quantity:    quantity,
			Meso:        meso,
			PetSlot:     petSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reservedEventStatusProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, dropId uint32, characterId uint32, itemId uint32, equipmentId uint32, quantity uint32, meso uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventReservedBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypeReserved,
		Body: messageDropKafka.StatusEventReservedBody{
			CharacterId: characterId,
			ItemId:      itemId,
			EquipmentId: equipmentId,
			Quantity:    quantity,
			Meso:        meso,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reservationFailureEventStatusProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventReservationFailureBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypeReservationFailure,
		Body: messageDropKafka.StatusEventReservationFailureBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
