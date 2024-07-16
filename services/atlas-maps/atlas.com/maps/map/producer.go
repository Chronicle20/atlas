package _map

import (
	"atlas-maps/kafka"
	"atlas-maps/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitEnterMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicMapStatus))
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		event := &statusEvent[characterEnter]{
			Tenant:    tenant,
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
			Type:      EventTopicMapStatusTypeCharacterEnter,
			Body: characterEnter{
				CharacterId: characterId,
			},
		}
		p(producer.CreateKey(int(mapId)), event)
	}
}

func emitExitMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicMapStatus))
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		event := &statusEvent[characterExit]{
			Tenant:    tenant,
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
			Type:      EventTopicMapStatusTypeCharacterExit,
			Body: characterExit{
				CharacterId: characterId,
			},
		}
		p(producer.CreateKey(int(mapId)), event)
	}
}
