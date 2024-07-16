package character

import (
	"atlas-character/kafka"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitStatusEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, name string, worldId byte, channelId byte, mapId uint32, eventType string) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicCharacterStatus))
	return func(characterId uint32, name string, worldId byte, channelId byte, mapId uint32, eventType string) {
		event := &statusEvent{
			Tenant:      tenant,
			CharacterId: characterId,
			Name:        name,
			WorldId:     worldId,
			ChannelId:   channelId,
			MapId:       mapId,
			Type:        eventType,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitCreatedEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, name string) {
	return func(characterId uint32, worldId byte, name string) {
		emitStatusEvent(l, span, tenant)(characterId, name, worldId, 0, 0, EventCharacterStatusTypeCreated)
	}
}

func emitLoginEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
		emitStatusEvent(l, span, tenant)(characterId, name, worldId, channelId, mapId, EventCharacterStatusTypeLogin)
	}
}

func emitLogoutEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
		emitStatusEvent(l, span, tenant)(characterId, name, worldId, channelId, mapId, EventCharacterStatusTypeLogout)
	}
}
