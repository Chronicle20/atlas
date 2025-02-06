package reactor

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/map/reactor"
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

func GetInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
			return InMapModelProvider(l)(ctx)(worldId, channelId, mapId)()
		}
	}
}

func doesNotExist(existing []Model) model.Filter[reactor.Model] {
	return func(reference reactor.Model) bool {
		for _, er := range existing {
			if er.Classification() == reference.Classification() && er.X() == reference.X() && er.Y() == reference.Y() {
				return false
			}
		}
		return true
	}
}

func Spawn(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32) error {
			existing, err := GetInMap(l)(ctx)(worldId, channelId, mapId)
			if err != nil {
				return err
			}

			np := model.FilteredProvider(reactor.InMapProvider(l)(ctx)(mapId), model.Filters[reactor.Model](doesNotExist(existing)))

			return model.ForEachSlice(np, IssueCreate(l)(ctx)(worldId, channelId, mapId), model.ParallelExecute())
		}
	}
}

func IssueCreate(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Operator[reactor.Model] {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Operator[reactor.Model] {
		return func(worldId byte, channelId byte, mapId uint32) model.Operator[reactor.Model] {
			return func(r reactor.Model) error {
				return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createCommandProvider(worldId, channelId, mapId, r.Classification(), r.Name(), 0, r.X(), r.Y(), r.Delay(), r.Direction()))
			}
		}
	}
}
