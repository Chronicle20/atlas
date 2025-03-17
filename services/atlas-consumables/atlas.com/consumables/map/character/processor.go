package character

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

func GetMap(characterId uint32) (_map.Model, error) {
	mk, ok := getRegistry().GetMap(characterId)
	if !ok {
		return _map.Model{}, errors.New("not found")
	}
	m := _map.NewModel(world.Id(mk.WorldId))(channel.Id(mk.ChannelId))(_map.Id(mk.MapId))
	return m, nil
}

func Enter(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		t := tenant.MustFromContext(ctx)
		getRegistry().AddCharacter(MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId}, characterId)
	}
}

func Exit(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().RemoveCharacter(characterId)
	}
}

func TransitionMap(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		Enter(ctx)(worldId, channelId, mapId, characterId)
	}
}

func TransitionChannel(ctx context.Context) func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
	return func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
		Enter(ctx)(worldId, channelId, mapId, characterId)
	}
}
