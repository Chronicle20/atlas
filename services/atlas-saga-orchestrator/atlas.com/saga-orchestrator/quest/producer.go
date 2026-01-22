package quest

import (
	"atlas-saga-orchestrator/kafka/message/quest"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func StartQuestCommandProvider(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, npcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest.Command[quest.StartCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          quest.CommandTypeStart,
		Body: quest.StartCommandBody{
			QuestId: questId,
			NpcId:   npcId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CompleteQuestCommandProvider(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest.Command[quest.CompleteCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          quest.CommandTypeComplete,
		Body: quest.CompleteCommandBody{
			QuestId:   questId,
			NpcId:     npcId,
			Selection: selection,
			Force:     force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ForfeitQuestCommandProvider(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest.Command[quest.ForfeitCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          quest.CommandTypeForfeit,
		Body: quest.ForfeitCommandBody{
			QuestId: questId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UpdateProgressCommandProvider(transactionId uuid.UUID, worldId byte, characterId uint32, questId uint32, infoNumber uint32, progress string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest.Command[quest.UpdateProgressCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          quest.CommandTypeUpdateProgress,
		Body: quest.UpdateProgressCommandBody{
			QuestId:    questId,
			InfoNumber: infoNumber,
			Progress:   progress,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
