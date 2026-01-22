package quest

import (
	quest2 "atlas-quest/kafka/message/quest"
	"context"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func QuestStartedEventProvider(transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestStartedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          quest2.StatusEventTypeStarted,
		Body: quest2.QuestStartedEventBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuestCompletedEventProvider(transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestCompletedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          quest2.StatusEventTypeCompleted,
		Body: quest2.QuestCompletedEventBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuestForfeitedEventProvider(transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestForfeitedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          quest2.StatusEventTypeForfeited,
		Body: quest2.QuestForfeitedEventBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuestProgressUpdatedEventProvider(transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32, infoNumber uint32, progress string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestProgressUpdatedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          quest2.StatusEventTypeProgressUpdated,
		Body: quest2.QuestProgressUpdatedEventBody{
			QuestId:    questId,
			InfoNumber: infoNumber,
			Progress:   progress,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ErrorStatusEventProvider(characterId uint32, worldId byte, questId uint32, errorType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.ErrorStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        quest2.StatusEventTypeError,
		Body: quest2.ErrorStatusEventBody{
			QuestId: questId,
			Error:   errorType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// emitEvent is a helper to emit quest status events
func emitEvent(l logrus.FieldLogger, ctx context.Context, provider model.Provider[[]kafka.Message]) error {
	sd := producer.SpanHeaderDecorator(ctx)
	td := producer.TenantHeaderDecorator(ctx)
	return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(quest2.EnvStatusEventTopic)))(sd, td)(provider)
}

// EmitQuestStarted emits a quest started event
func EmitQuestStarted(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32) error {
	return emitEvent(l, ctx, QuestStartedEventProvider(transactionId, characterId, worldId, questId))
}

// EmitQuestCompleted emits a quest completed event
func EmitQuestCompleted(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32) error {
	return emitEvent(l, ctx, QuestCompletedEventProvider(transactionId, characterId, worldId, questId))
}

// EmitQuestForfeited emits a quest forfeited event
func EmitQuestForfeited(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32) error {
	return emitEvent(l, ctx, QuestForfeitedEventProvider(transactionId, characterId, worldId, questId))
}

// EmitProgressUpdated emits a quest progress updated event
func EmitProgressUpdated(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, worldId byte, questId uint32, infoNumber uint32, progress string) error {
	return emitEvent(l, ctx, QuestProgressUpdatedEventProvider(transactionId, characterId, worldId, questId, infoNumber, progress))
}
