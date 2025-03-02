package messenger

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &commandEvent[createCommandBody]{
		ActorId: leaderId,
		Type:    CommandMessengerCreate,
		Body:    createCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinCommandProvider(messengerId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[joinCommandBody]{
		ActorId: characterId,
		Type:    CommandMessengerJoin,
		Body: joinCommandBody{
			MessengerId: messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(messengerId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[leaveCommandBody]{
		ActorId: characterId,
		Type:    CommandMessengerLeave,
		Body: leaveCommandBody{
			MessengerId: messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider(actorId uint32, messengerId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &statusEvent[createdEventBody]{
		ActorId:     actorId,
		MessengerId: messengerId,
		WorldId:     worldId,
		Type:        EventMessengerStatusTypeCreated,
		Body:        createdEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinedEventProvider(actorId uint32, messengerId uint32, worldId byte, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &statusEvent[joinedEventBody]{
		ActorId:     actorId,
		MessengerId: messengerId,
		WorldId:     worldId,
		Type:        EventMessengerStatusTypeJoined,
		Body: joinedEventBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leftEventProvider(actorId uint32, messengerId uint32, worldId byte, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &statusEvent[leftEventBody]{
		ActorId:     actorId,
		MessengerId: messengerId,
		WorldId:     worldId,
		Type:        EventMessengerStatusTypeLeft,
		Body: leftEventBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorEventProvider(actorId uint32, messengerId uint32, worldId byte, errorType string, characterName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &statusEvent[errorEventBody]{
		ActorId:     actorId,
		MessengerId: messengerId,
		WorldId:     worldId,
		Type:        EventMessengerStatusTypeError,
		Body: errorEventBody{
			Type:          errorType,
			CharacterName: characterName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
