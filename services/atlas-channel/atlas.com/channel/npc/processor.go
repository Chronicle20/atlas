package npc

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ForEachInMap(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, f model.Operator[Model]) error {
	return func(ctx context.Context) func(mapId uint32, f model.Operator[Model]) error {
		return func(mapId uint32, f model.Operator[Model]) error {
			return model.ForEachSlice(InMapModelProvider(l)(ctx)(mapId), f, model.ParallelExecute())
		}
	}
}

func InMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
		return func(mapId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestNPCsInMap(mapId), Extract, model.Filters[Model]())
		}
	}
}

func InMapByObjectIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, objectId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32, objectId uint32) model.Provider[[]Model] {
		return func(mapId uint32, objectId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestNPCsInMapByObjectId(mapId, objectId), Extract, model.Filters[Model]())
		}
	}
}

func GetInMapByObjectId(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, objectId uint32) (Model, error) {
	return func(ctx context.Context) func(mapId uint32, objectId uint32) (Model, error) {
		return func(mapId uint32, objectId uint32) (Model, error) {
			p := InMapByObjectIdModelProvider(l)(ctx)(mapId, objectId)
			return model.First[Model](p, model.Filters[Model]())
		}
	}
}
