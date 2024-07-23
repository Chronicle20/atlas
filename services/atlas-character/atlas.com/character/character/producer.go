package character

import (
	"atlas-character/kafka"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitCreatedEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, name string) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicCharacterStatus))
	return func(characterId uint32, worldId byte, name string) {
		event := &statusEvent[statusEventCreatedBody]{
			Tenant:      tenant,
			CharacterId: characterId,
			WorldId:     worldId,
			Type:        EventCharacterStatusTypeCreated,
			Body: statusEventCreatedBody{
				Name: name,
			},
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitLoginEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicCharacterStatus))
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
		event := &statusEvent[statusEventLoginBody]{
			Tenant:      tenant,
			CharacterId: characterId,
			WorldId:     worldId,
			Type:        EventCharacterStatusTypeLogin,
			Body: statusEventLoginBody{
				ChannelId: channelId,
				MapId:     mapId,
			},
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitLogoutEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicCharacterStatus))
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) {
		event := &statusEvent[statusEventLogoutBody]{
			Tenant:      tenant,
			CharacterId: characterId,
			WorldId:     worldId,
			Type:        EventCharacterStatusTypeLogout,
			Body: statusEventLogoutBody{
				ChannelId: channelId,
				MapId:     mapId,
			},
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitMapChangedEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicCharacterStatus))
	return func(characterId uint32, worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) {
		event := &statusEvent[statusEventMapChangedBody]{
			Tenant:      tenant,
			CharacterId: characterId,
			WorldId:     worldId,
			Type:        EventCharacterStatusTypeLogout,
			Body: statusEventMapChangedBody{
				ChannelId:      channelId,
				OldMapId:       oldMapId,
				TargetMapId:    targetMapId,
				TargetPortalId: targetPortalId,
			},
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}
