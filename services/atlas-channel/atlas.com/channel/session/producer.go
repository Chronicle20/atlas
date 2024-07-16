package session

import (
	"atlas-channel/kafka"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitStatusEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte, eventType string) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicSessionStatus))
	return func(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte, eventType string) {
		event := &statusEvent{
			Tenant:      tenant,
			SessionId:   sessionId,
			AccountId:   accountId,
			CharacterId: characterId,
			WorldId:     worldId,
			ChannelId:   channelId,
			Issuer:      EventSessionStatusIssuerChannel,
			Type:        eventType,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitCreatedStatusEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte) {
	return func(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte) {
		emitStatusEvent(l, span, tenant)(sessionId, accountId, characterId, worldId, channelId, EventSessionStatusTypeCreated)
	}
}

func emitDestroyedStatusEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte) {
	return func(sessionId uuid.UUID, accountId uint32, characterId uint32, worldId byte, channelId byte) {
		emitStatusEvent(l, span, tenant)(sessionId, accountId, characterId, worldId, channelId, EventSessionStatusTypeDestroyed)
	}
}
