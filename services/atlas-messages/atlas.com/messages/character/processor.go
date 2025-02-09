package character

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
		}
	}
}

func byNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(name string) model.Provider[[]Model] {
	return func(ctx context.Context) func(name string) model.Provider[[]Model] {
		return func(name string) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByName(name), Extract, model.Filters[Model]())
		}
	}
}

func GetByName(l logrus.FieldLogger) func(ctx context.Context) func(name string) (Model, error) {
	return func(ctx context.Context) func(name string) (Model, error) {
		return func(name string) (Model, error) {
			return model.First(byNameProvider(l)(ctx)(name), model.Filters[Model]())
		}
	}
}

func IdByNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(name string) model.Provider[uint32] {
	return func(ctx context.Context) func(name string) model.Provider[uint32] {
		return func(name string) model.Provider[uint32] {
			c, err := GetByName(l)(ctx)(name)
			if err != nil {
				return model.ErrorProvider[uint32](err)
			}
			return model.FixedProvider(c.Id())
		}
	}
}
