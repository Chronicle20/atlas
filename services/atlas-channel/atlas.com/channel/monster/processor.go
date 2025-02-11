package monster

import (
	"atlas-channel/kafka/producer"
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

func Damage(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, monsterId uint32, characterId uint32, damage uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, monsterId uint32, characterId uint32, damage uint32) error {
		return func(worldId byte, channelId byte, monsterId uint32, characterId uint32, damage uint32) error {
			l.Debugf("Applying damage to monster [%d]. Character [%d]. Damage [%d].", monsterId, characterId, damage)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(damageCommandProvider(worldId, channelId, monsterId, characterId, damage))
		}
	}
}
