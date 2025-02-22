package monster

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
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

func GetInMap(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model) ([]Model, error) {
	return func(ctx context.Context) func(m _map.Model) ([]Model, error) {
		return func(m _map.Model) ([]Model, error) {
			return InMapModelProvider(l)(ctx)(m)()
		}
	}
}

func Damage(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, monsterId uint32, characterId uint32, damage uint32) error {
	return func(ctx context.Context) func(m _map.Model, monsterId uint32, characterId uint32, damage uint32) error {
		return func(m _map.Model, monsterId uint32, characterId uint32, damage uint32) error {
			l.Debugf("Applying damage to monster [%d]. Character [%d]. Damage [%d].", monsterId, characterId, damage)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(damageCommandProvider(m, monsterId, characterId, damage))
		}
	}
}
