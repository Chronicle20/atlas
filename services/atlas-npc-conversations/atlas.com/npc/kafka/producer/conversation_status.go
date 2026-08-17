package producer

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// StartedStatusProvider reports that a saga-driven conversation opened. Keyed
// by character id, matching every other producer on the conversation topics.
func StartedStatusProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32, sourceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationStatusEvent[npc2.StatusEventStartedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          npc2.StatusEventTypeStarted,
		Body: npc2.StatusEventStartedBody{
			NpcTemplateId: npcTemplateId,
			SourceId:      sourceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// StartErrorStatusProvider reports that a saga-driven conversation did not
// open. The awaiting step fails, which fails the saga, which means the
// following destroy step never runs — the player keeps the item.
func StartErrorStatusProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32, sourceId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.ConversationStatusEvent[npc2.StatusEventStartErrorBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          npc2.StatusEventTypeStartError,
		Body: npc2.StatusEventStartErrorBody{
			NpcTemplateId: npcTemplateId,
			SourceId:      sourceId,
			Reason:        reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
