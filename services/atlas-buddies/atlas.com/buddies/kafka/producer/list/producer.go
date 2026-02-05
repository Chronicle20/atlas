package list

import (
	list2 "atlas-buddies/kafka/message/list"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(characterId character.Id, capacity byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.Command[list2.CreateCommandBody]{
		CharacterId: characterId,
		Type:        list2.CommandTypeCreate,
		Body: list2.CreateCommandBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuddyAddedStatusEventProvider(characterId character.Id, worldId world.Id, buddyId character.Id, buddyName string, buddyChannelId int8, group string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.BuddyAddedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeBuddyAdded,
		Body: list2.BuddyAddedStatusEventBody{
			CharacterId:   buddyId,
			Group:         group,
			CharacterName: buddyName,
			ChannelId:     buddyChannelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuddyRemovedStatusEventProvider(characterId character.Id, worldId world.Id, buddyId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.BuddyRemovedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeBuddyRemoved,
		Body: list2.BuddyRemovedStatusEventBody{
			CharacterId: buddyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuddyUpdatedStatusEventProvider(characterId character.Id, worldId world.Id, buddyId character.Id, group string, buddyName string, channelId int8, inShop bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.BuddyUpdatedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeBuddyUpdated,
		Body: list2.BuddyUpdatedStatusEventBody{
			CharacterId:   buddyId,
			Group:         group,
			CharacterName: buddyName,
			ChannelId:     channelId,
			InShop:        inShop,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuddyChannelChangeStatusEventProvider(characterId character.Id, worldId world.Id, buddyId character.Id, buddyChannelId int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.BuddyChannelChangeStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeBuddyChannelChange,
		Body: list2.BuddyChannelChangeStatusEventBody{
			CharacterId: buddyId,
			ChannelId:   buddyChannelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuddyCapacityChangeStatusEventProvider(characterId character.Id, worldId world.Id, capacity byte) model.Provider[[]kafka.Message] {
	return BuddyCapacityChangeStatusEventWithTransactionProvider(characterId, worldId, capacity, uuid.Nil)
}

func BuddyCapacityChangeStatusEventWithTransactionProvider(characterId character.Id, worldId world.Id, capacity byte, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.BuddyCapacityChangeStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeBuddyCapacityUpdate,
		Body: list2.BuddyCapacityChangeStatusEventBody{
			Capacity:      capacity,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ErrorStatusEventProvider(characterId character.Id, worldId world.Id, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.ErrorStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeError,
		Body: list2.ErrorStatusEventBody{
			Error: error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
