package drop

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func InMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model) model.Provider[[]Model] {
	return func(ctx context.Context) func(m _map.Model) model.Provider[[]Model] {
		return func(m _map.Model) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMap(m), Extract, model.Filters[Model]())
		}
	}
}

func ForEachInMap(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, f model.Operator[Model]) error {
	return func(ctx context.Context) func(m _map.Model, f model.Operator[Model]) error {
		return func(m _map.Model, f model.Operator[Model]) error {
			return model.ForEachSlice(InMapModelProvider(l)(ctx)(m), f, model.ParallelExecute())
		}
	}
}

func RequestReservation(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, dropId uint32, characterId uint32, characterX int16, characterY int16, petSlot int8) error {
	return func(ctx context.Context) func(m _map.Model, dropId uint32, characterId uint32, characterX int16, characterY int16, petSlot int8) error {
		return func(m _map.Model, dropId uint32, characterId uint32, characterX int16, characterY int16, petSlot int8) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestReservationCommandProvider(m, dropId, characterId, characterX, characterY, petSlot))
		}
	}
}
