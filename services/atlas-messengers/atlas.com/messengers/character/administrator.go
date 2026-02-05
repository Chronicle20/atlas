package character

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
)

func CreateCharacter(ctx context.Context) func(ch channel.Model, characterId uint32, name string) Model {
	return func(ch channel.Model, characterId uint32, name string) Model {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().Create(t, ch, characterId, name)
	}
}

func UpdateCharacter(ctx context.Context) func(characterId uint32, updaters ...func(Model) Model) Model {
	return func(characterId uint32, updaters ...func(Model) Model) Model {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().Update(t, characterId, updaters...)
	}
}
