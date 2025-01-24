package portal

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByNamePortalIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, name string) model.Provider[uint32] {
	return func(ctx context.Context) func(mapId uint32, name string) model.Provider[uint32] {
		return func(mapId uint32, name string) model.Provider[uint32] {
			return model.Map[Model, uint32](func(m Model) (uint32, error) {
				return m.Id(), nil
			})(ByNameProvider(l)(ctx)(mapId, name))
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
			return model.Map[Model, uint32](func(m Model) (uint32, error) {
				return m.Id(), nil
			})(RandomPortalProvider(l)(ctx)(mapId))
		}
	}
}

func InMapProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
		return func(mapId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestAll(mapId), Extract, model.Filters[Model]())
		}
	}
}

func ByNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, portalName string) model.Provider[Model] {
	return func(ctx context.Context) func(mapId uint32, portalName string) model.Provider[Model] {
		return func(mapId uint32, portalName string) model.Provider[Model] {
			sp := requests.SliceProvider[RestModel, Model](l, ctx)(requestInMapByName(mapId, portalName), Extract, model.Filters[Model]())
			return model.FirstProvider(sp, model.Filters[Model]())
		}
	}
}
