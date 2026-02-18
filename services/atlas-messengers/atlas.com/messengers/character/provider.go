package character

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
)

func ByIdProvider(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return func() (Model, error) {
			return GetRegistry().Get(ctx, characterId)
		}
	}
}
