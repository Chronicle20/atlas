package drop

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventStatusProvider(drop Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(drop.Id()))
	value := &statusEvent[createdStatusEventBody]{
		WorldId:   drop.WorldId(),
		ChannelId: drop.ChannelId(),
		MapId:     drop.MapId(),
		DropId:    drop.Id(),
		Type:      StatusEventTypeCreated,
		Body: createdStatusEventBody{
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

func expiredEventStatusProvider(worldId byte, channelId byte, mapId uint32, dropId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &statusEvent[expiredStatusEventBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		DropId:    dropId,
		Type:      StatusEventTypeExpired,
		Body:      expiredStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func pickedUpEventStatusProvider(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &statusEvent[pickedUpStatusEventBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		DropId:    dropId,
		Type:      StatusEventTypePickedUp,
		Body: pickedUpStatusEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reservedEventStatusProvider(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &statusEvent[reservedStatusEventBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		DropId:    dropId,
		Type:      StatusEventTypeReserved,
		Body: reservedStatusEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reservationFailureEventStatusProvider(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &statusEvent[reservationFailureStatusEventBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		DropId:    dropId,
		Type:      StatusEventTypeReservationFailure,
		Body: reservationFailureStatusEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
