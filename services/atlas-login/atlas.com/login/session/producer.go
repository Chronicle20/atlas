package session

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func statusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte, eventType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent{
		SessionId:   sessionId,
		AccountId:   accountId,
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Issuer:      EventSessionStatusIssuerLogin,
		Type:        eventType,
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusEventProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(sessionId, accountId, 0, 0, 0, EventSessionStatusTypeCreated)
}

func destroyedStatusEventProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(sessionId, accountId, 0, 0, 0, EventSessionStatusTypeDestroyed)
}
