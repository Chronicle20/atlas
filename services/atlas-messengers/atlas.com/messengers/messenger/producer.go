package messenger

import (
	"atlas-messengers/kafka/message/messenger"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(transactionID uuid.UUID, leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &messenger.CommandEvent[messenger.CreateCommandBody]{
		TransactionID: transactionID,
		ActorId:       leaderId,
		Type:          messenger.CommandMessengerCreate,
		Body:          messenger.CreateCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinCommandProvider(transactionID uuid.UUID, messengerId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &messenger.CommandEvent[messenger.JoinCommandBody]{
		TransactionID: transactionID,
		ActorId:       characterId,
		Type:          messenger.CommandMessengerJoin,
		Body: messenger.JoinCommandBody{
			MessengerId: messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(transactionID uuid.UUID, messengerId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &messenger.CommandEvent[messenger.LeaveCommandBody]{
		TransactionID: transactionID,
		ActorId:       characterId,
		Type:          messenger.CommandMessengerLeave,
		Body: messenger.LeaveCommandBody{
			MessengerId: messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider(transactionID uuid.UUID, actorId uint32, messengerId uint32, worldId world.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &messenger.StatusEvent[messenger.CreatedEventBody]{
		TransactionID: transactionID,
		ActorId:       actorId,
		MessengerId:   messengerId,
		WorldId:       worldId,
		Type:          messenger.EventMessengerStatusTypeCreated,
		Body:          messenger.CreatedEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinedEventProvider(transactionID uuid.UUID, actorId uint32, messengerId uint32, worldId world.Id, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &messenger.StatusEvent[messenger.JoinedEventBody]{
		TransactionID: transactionID,
		ActorId:       actorId,
		MessengerId:   messengerId,
		WorldId:       worldId,
		Type:          messenger.EventMessengerStatusTypeJoined,
		Body: messenger.JoinedEventBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leftEventProvider(transactionID uuid.UUID, actorId uint32, messengerId uint32, worldId world.Id, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &messenger.StatusEvent[messenger.LeftEventBody]{
		TransactionID: transactionID,
		ActorId:       actorId,
		MessengerId:   messengerId,
		WorldId:       worldId,
		Type:          messenger.EventMessengerStatusTypeLeft,
		Body: messenger.LeftEventBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorEventProvider(transactionID uuid.UUID, actorId uint32, messengerId uint32, worldId world.Id, errorType string, characterName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &messenger.StatusEvent[messenger.ErrorEventBody]{
		TransactionID: transactionID,
		ActorId:       actorId,
		MessengerId:   messengerId,
		WorldId:       worldId,
		Type:          messenger.EventMessengerStatusTypeError,
		Body: messenger.ErrorEventBody{
			Type:          errorType,
			CharacterName: characterName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
