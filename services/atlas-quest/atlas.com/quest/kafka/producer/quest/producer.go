package quest

import (
	quest2 "atlas-quest/kafka/message/quest"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func QuestStartedEventProvider(characterId uint32, worldId byte, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestStartedEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        quest2.StatusEventTypeStarted,
		Body: quest2.QuestStartedEventBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuestCompletedEventProvider(characterId uint32, worldId byte, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestCompletedEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        quest2.StatusEventTypeCompleted,
		Body: quest2.QuestCompletedEventBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuestForfeitedEventProvider(characterId uint32, worldId byte, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestForfeitedEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        quest2.StatusEventTypeForfeited,
		Body: quest2.QuestForfeitedEventBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuestProgressUpdatedEventProvider(characterId uint32, worldId byte, questId uint32, infoNumber uint32, progress string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.StatusEvent[quest2.QuestProgressUpdatedEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        quest2.StatusEventTypeProgressUpdated,
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
