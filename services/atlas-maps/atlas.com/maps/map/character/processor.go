package character

import (
	"atlas-maps/tenant"
)

func GetCharactersInMap(tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return getRegistry().GetInMap(MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}), nil
	}
}

func GetMapsWithCharacters() []MapKey {
	return getRegistry().GetMapsWithCharacters()
}

func Enter(tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().AddCharacter(MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}

func Exit(tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().RemoveCharacter(MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}
