package chalkboard

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

func AttemptUse(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, message string) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, message string) error {
		return func(m _map.Model, characterId uint32, message string) error {
			l.Debugf("Character [%d] attempting to set a chalkboard message [%s].", characterId, message)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(setCommandProvider(m, characterId, message))
		}
	}
}

func Close(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32) error {
		return func(m _map.Model, characterId uint32) error {
			l.Debugf("Character [%d] attempting to close chalkboard.", characterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(clearCommandProvider(m, characterId))
		}
	}
}
