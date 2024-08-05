package character

import (
	"atlas-maps/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetCharactersInMap(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return getRegistry().GetInMap(MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}), nil
	}
}

func GetMapsWithCharacters(_ logrus.FieldLogger, _ opentracing.Span) []MapKey {
	return getRegistry().GetMapsWithCharacters()
}

func Enter(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().AddCharacter(MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}

func Exit(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().RemoveCharacter(MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}
