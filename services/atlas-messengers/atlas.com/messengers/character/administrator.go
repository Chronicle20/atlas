package character

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

func CreateCharacter(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, name string) Model {
	return func(worldId world.Id, channelId channel.Id, characterId uint32, name string) Model {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().Create(t, worldId, channelId, characterId, name)
	}
}

func UpdateCharacter(ctx context.Context) func(characterId uint32, updaters ...func(Model) Model) Model {
	return func(characterId uint32, updaters ...func(Model) Model) Model {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().Update(t, characterId, updaters...)
	}
}
