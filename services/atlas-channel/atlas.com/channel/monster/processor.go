package monster

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(uniqueId uint32) (Model, error) {
	return func(ctx context.Context) func(uniqueId uint32) (Model, error) {
		return func(uniqueId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(uniqueId), Extract)()
		}
	}
}

func InMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMap(worldId, channelId, mapId), Extract, model.Filters[Model]())
		}
	}
}

func ForEachInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
		return func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
			return model.ForEachSlice(InMapModelProvider(l)(ctx)(worldId, channelId, mapId), f, model.ParallelExecute())
		}
	}
}

func GetInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
			return InMapModelProvider(l)(ctx)(worldId, channelId, mapId)()
		}
	}
}
