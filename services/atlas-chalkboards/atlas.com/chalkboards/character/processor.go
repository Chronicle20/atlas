package character

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
)

func InMapProvider(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
		t := tenant.MustFromContext(ctx)
		cids := getRegistry().GetInMap(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId})
		return model.FixedProvider(cids)
	}
}

func GetCharactersInMap(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return InMapProvider(ctx)(worldId, channelId, mapId)()
	}
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

func TransitionMap(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		Exit(ctx)(worldId, channelId, oldMapId, characterId)
		Enter(ctx)(worldId, channelId, mapId, characterId)
	}
}

func TransitionChannel(ctx context.Context) func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
	return func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
		Exit(ctx)(worldId, oldChannelId, mapId, characterId)
		Enter(ctx)(worldId, channelId, mapId, characterId)
	}
}
