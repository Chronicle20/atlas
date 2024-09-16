package monster

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(uniqueId uint32) (Model, error) {
	return func(uniqueId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(ctx, tenant)(uniqueId), Extract)()
	}
}

func InMapModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestInMap(ctx, tenant)(worldId, channelId, mapId), Extract)
	}
}

func ForEachInMap(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
	return func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
		return model.ForEachSlice(InMapModelProvider(l, ctx, tenant)(worldId, channelId, mapId), f, model.ParallelExecute())
	}
}

func GetInMap(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		return InMapModelProvider(l, ctx, tenant)(worldId, channelId, mapId)()
	}
}
