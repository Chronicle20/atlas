package _map

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	"atlas-maps/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Transition(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		Exit(l, span, tenant)(worldId, channelId, oldMapId, characterId)
		Enter(l, span, tenant)(worldId, channelId, mapId, characterId)
	}
}

func Enter(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		character.Enter(l, span, tenant)(worldId, channelId, mapId, characterId)
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicMapStatus)(enterMapProvider(tenant, worldId, channelId, mapId, characterId))
	}
}

func Exit(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		character.Exit(l, span, tenant)(worldId, channelId, mapId, characterId)
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicMapStatus)(exitMapProvider(tenant, worldId, channelId, mapId, characterId))
	}
}
