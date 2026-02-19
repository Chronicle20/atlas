package messenger

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
)

func ByIdProvider(ctx context.Context) func(messengerId uint32) model.Provider[Model] {
	return func(messengerId uint32) model.Provider[Model] {
		return func() (Model, error) {
			return GetRegistry().Get(ctx, messengerId)
		}
	}
}

func GetAllProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetRegistry().GetAll(ctx), nil
	}
}
