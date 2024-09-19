package portal

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func InMapProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
		return func(mapId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestAll(mapId), Extract, model.Filters[Model]())
		}
	}
}

func RandomPortalProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[Model] {
		return func(mapId uint32) model.Provider[Model] {
			return func() (Model, error) {
				ps, err := InMapProvider(l)(ctx)(mapId)()
				if err != nil {
					return Model{}, err
				}
				return model.RandomPreciselyOneFilter(ps)
			}
		}
	}
}

func RandomPortalIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[uint32] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[uint32] {
		return func(mapId uint32) model.Provider[uint32] {
			return model.Map(getId)(RandomPortalProvider(l)(ctx)(mapId))
		}
	}
}

func getId(m Model) (uint32, error) {
	return m.Id(), nil
}
