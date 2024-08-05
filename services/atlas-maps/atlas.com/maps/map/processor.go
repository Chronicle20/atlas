package _map

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetCharactersInMap(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return getRegistry().GetInMap(MapKey{TenantId: tenant.Id, WorldId: worldId, ChannelId: channelId, MapId: mapId}), nil
	}
}

func Transition(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		Exit(l, span, tenant)(worldId, channelId, oldMapId, characterId)
		Enter(l, span, tenant)(worldId, channelId, mapId, characterId)
	}
}

func Enter(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().AddCharacter(MapKey{TenantId: tenant.Id, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicMapStatus)(enterMapProvider(tenant, worldId, channelId, mapId, characterId))
	}
}

func Exit(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().RemoveCharacter(MapKey{TenantId: tenant.Id, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicMapStatus)(exitMapProvider(tenant, worldId, channelId, mapId, characterId))
	}
}
