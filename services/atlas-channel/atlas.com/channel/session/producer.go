package session

import (
	session2 "atlas-channel/kafka/message/session"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func StatusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, ch channel.Model, eventType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &session2.StatusEvent{
		SessionId:   sessionId,
		AccountId:   accountId,
		CharacterId: characterId,
		WorldId:     ch.WorldId(),
		ChannelId:   ch.Id(),
		Issuer:      session2.EventSessionStatusIssuerChannel,
		Type:        eventType,
	}
	return producer.SingleMessageProvider(key, value)
}

func CreatedStatusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, ch channel.Model) model.Provider[[]kafka.Message] {
	return StatusEventProvider(sessionId, accountId, characterId, ch, session2.EventSessionStatusTypeCreated)
}

func DestroyedStatusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, ch channel.Model) model.Provider[[]kafka.Message] {
	return StatusEventProvider(sessionId, accountId, characterId, ch, session2.EventSessionStatusTypeDestroyed)
}
