package character

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
)

func GetCharactersInMap(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		t := tenant.MustFromContext(ctx)
		return getRegistry().GetInMap(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}), nil
	}
}

func GetMapsWithCharacters() []MapKey {
	return getRegistry().GetMapsWithCharacters()
}

func Enter(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		t := tenant.MustFromContext(ctx)
		getRegistry().AddCharacter(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}

func Exit(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		t := tenant.MustFromContext(ctx)
		getRegistry().RemoveCharacter(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}
