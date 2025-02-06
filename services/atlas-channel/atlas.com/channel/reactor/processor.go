package reactor

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func InMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMap(worldId, channelId, mapId), Extract, model.Filters[Model]())
		}
	}
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:             rm.Id,
		worldId:        rm.WorldId,
		channelId:      rm.ChannelId,
		mapId:          rm.MapId,
		classification: rm.Classification,
		name:           rm.Name,
		state:          rm.State,
		eventState:     rm.EventState,
		delay:          rm.Delay,
		direction:      rm.Direction,
		x:              rm.X,
		y:              rm.Y,
	}, nil
}

func ForEachInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
		return func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
			return model.ForEachSlice(InMapModelProvider(l)(ctx)(worldId, channelId, mapId), f, model.ParallelExecute())
		}
	}
}
