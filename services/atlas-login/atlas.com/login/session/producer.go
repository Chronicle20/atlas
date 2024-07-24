package session

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func statusEventProvider(tenant tenant.Model, sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte, eventType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent{
		Tenant:      tenant,
		SessionId:   sessionId,
		AccountId:   accountId,
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Issuer:      EventSessionStatusIssuerChannel,
		Type:        eventType,
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusEventProvider(tenant tenant.Model, sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(tenant, sessionId, accountId, 0, 0, 0, EventSessionStatusTypeCreated)
}

func destroyedStatusEventProvider(tenant tenant.Model, sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(tenant, sessionId, accountId, 0, 0, 0, EventSessionStatusTypeDestroyed)
}
