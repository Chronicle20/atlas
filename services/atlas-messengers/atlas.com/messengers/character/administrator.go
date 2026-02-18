package character

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
)

func CreateCharacter(ctx context.Context) func(ch channel.Model, characterId uint32, name string) Model {
	return func(ch channel.Model, characterId uint32, name string) Model {
		return GetRegistry().Create(ctx, ch, characterId, name)
	}
}

func UpdateCharacter(ctx context.Context) func(characterId uint32, updaters ...func(Model) Model) Model {
	return func(characterId uint32, updaters ...func(Model) Model) Model {
		return GetRegistry().Update(ctx, characterId, updaters...)
	}
}
