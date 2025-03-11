package character

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}

func GetLoggedIn() model.Provider[map[uint32]MapKey] {
	return model.FixedProvider(getRegistry().GetLoggedIn())
}

func Enter(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		t := tenant.MustFromContext(ctx)
		getRegistry().AddCharacter(characterId, MapKey{Tenant: t, WorldId: worldId, ChannelId: channelId, MapId: mapId})
	}
}

func Exit(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		getRegistry().RemoveCharacter(characterId)
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
