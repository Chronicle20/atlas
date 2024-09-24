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

func RandomSpawnPointProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[Model] {
		return func(mapId uint32) model.Provider[Model] {
			return func() (Model, error) {
				sps, err := model.FilteredProvider(InMapProvider(l)(ctx)(mapId), model.Filters(ValidPortal, SpawnPoint, NoTarget))()
				if err != nil {
					return Model{}, err
				}
				return model.RandomPreciselyOneFilter(sps)
			}
		}
	}
}

func RandomSpawnPointIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[uint32] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[uint32] {
		return func(mapId uint32) model.Provider[uint32] {
			return model.Map(getId)(RandomSpawnPointProvider(l)(ctx)(mapId))
		}
	}
}

func getId(m Model) (uint32, error) {
	return m.Id(), nil
}
