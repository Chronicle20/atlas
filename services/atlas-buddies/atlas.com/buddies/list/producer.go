package list

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(characterId uint32, capacity byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[createCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeCreate,
		Body: createCommandBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func buddyAddedStatusEventProvider(characterId uint32, worldId byte, buddyId uint32, buddyName string, buddyChannelId int8, group string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[buddyAddedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeBuddyAdded,
		Body: buddyAddedStatusEventBody{
			CharacterId:   buddyId,
			Group:         group,
			CharacterName: buddyName,
			ChannelId:     buddyChannelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func buddyRemovedStatusEventProvider(characterId uint32, worldId byte, buddyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[buddyRemovedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeBuddyRemoved,
		Body: buddyRemovedStatusEventBody{
			CharacterId: buddyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func buddyChannelChangeStatusEventProvider(characterId uint32, worldId byte, buddyId uint32, buddyChannelId int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[buddyChannelChangeStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeBuddyChannelChange,
		Body: buddyChannelChangeStatusEventBody{
			CharacterId: buddyId,
			ChannelId:   buddyChannelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func buddyCapacityChangeStatusEventProvider(characterId uint32, worldId byte, capacity byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[buddyCapacityChangeStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeBuddyCapacityUpdate,
		Body: buddyCapacityChangeStatusEventBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorStatusEventProvider(characterId uint32, worldId byte, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[errorStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeError,
		Body: errorStatusEventBody{
			Error: error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
