package session

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func statusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId world.Id, channelId channel.Id, eventType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent{
		SessionId:   sessionId,
		AccountId:   accountId,
		CharacterId: characterId,
		WorldId:     byte(worldId),
		ChannelId:   byte(channelId),
		Issuer:      EventSessionStatusIssuerChannel,
		Type:        eventType,
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId world.Id, channelId channel.Id) model.Provider[[]kafka.Message] {
	return statusEventProvider(sessionId, accountId, characterId, worldId, channelId, EventSessionStatusTypeCreated)
}

func destroyedStatusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId world.Id, channelId channel.Id) model.Provider[[]kafka.Message] {
	return statusEventProvider(sessionId, accountId, characterId, worldId, channelId, EventSessionStatusTypeDestroyed)
}
