package drop

import (
	"atlas-channel/kafka/producer"
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

func ForEachInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
		return func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
			return model.ForEachSlice(InMapModelProvider(l)(ctx)(worldId, channelId, mapId), f, model.ParallelExecute())
		}
	}
}

func RequestReservation(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, characterX int16, characterY int16) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, characterX int16, characterY int16) error {
		return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, characterX int16, characterY int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestReservationCommandProvider(worldId, channelId, mapId, dropId, characterId, characterX, characterY))
		}
	}
}
