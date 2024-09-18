package data

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(id uint32) model.Provider[[]Model] {
		return func(id uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestById(id), Extract, model.Filters[Model]())
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) ([]Model, error) {
	return func(ctx context.Context) func(id uint32) ([]Model, error) {
		return func(id uint32) ([]Model, error) {
			return byIdModelProvider(l)(ctx)(id)()
		}
	}
}
