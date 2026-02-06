package messenger

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
)

func ByIdProvider(ctx context.Context) func(messengerId uint32) model.Provider[Model] {
	return func(messengerId uint32) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetRegistry().Get(t, messengerId)
		}
	}
}

func GetAllProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().GetAll(t), nil
	}
}
