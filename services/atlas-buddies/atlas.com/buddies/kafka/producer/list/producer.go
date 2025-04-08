package list

import (
	list2 "atlas-buddies/kafka/message/list"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(characterId uint32, capacity byte) model.Provider[[]kafka.Message] {
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

func BuddyAddedStatusEventProvider(characterId uint32, worldId byte, buddyId uint32, buddyName string, buddyChannelId int8, group string) model.Provider[[]kafka.Message] {
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

func BuddyRemovedStatusEventProvider(characterId uint32, worldId byte, buddyId uint32) model.Provider[[]kafka.Message] {
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

func BuddyUpdatedStatusEventProvider(characterId uint32, worldId byte, buddyId uint32, group string, buddyName string, channelId int8, inShop bool) model.Provider[[]kafka.Message] {
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

func BuddyChannelChangeStatusEventProvider(characterId uint32, worldId byte, buddyId uint32, buddyChannelId int8) model.Provider[[]kafka.Message] {
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

func BuddyCapacityChangeStatusEventProvider(characterId uint32, worldId byte, capacity byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &list2.StatusEvent[list2.BuddyCapacityChangeStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        list2.StatusEventTypeBuddyCapacityUpdate,
		Body: list2.BuddyCapacityChangeStatusEventBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ErrorStatusEventProvider(characterId uint32, worldId byte, error string) model.Provider[[]kafka.Message] {
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
