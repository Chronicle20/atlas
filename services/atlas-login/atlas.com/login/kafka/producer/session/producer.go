package session

import (
	"atlas-login/kafka/message/session"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func StatusEventProvider(sessionId uuid.UUID, accountId uint32, characterId uint32, ch channel.Model, eventType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &session.StatusEvent{
		SessionId:   sessionId,
		AccountId:   accountId,
		CharacterId: characterId,
		WorldId:     ch.WorldId(),
		ChannelId:   ch.Id(),
		Issuer:      session.EventSessionStatusIssuerLogin,
		Type:        eventType,
	}
	return producer.SingleMessageProvider(key, value)
}

func CreatedStatusEventProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	ch := channel.NewModel(0, 0)
	return StatusEventProvider(sessionId, accountId, 0, ch, session.EventSessionStatusTypeCreated)
}

func DestroyedStatusEventProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	ch := channel.NewModel(0, 0)
	return StatusEventProvider(sessionId, accountId, 0, ch, session.EventSessionStatusTypeDestroyed)
}
