package portal

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func inMapByNameModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, name string) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32, name string) model.Provider[[]Model] {
		return func(mapId uint32, name string) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
		}
	}
}

func GetInMapByName(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, name string) (Model, error) {
	return func(ctx context.Context) func(mapId uint32, name string) (Model, error) {
		return func(mapId uint32, name string) (Model, error) {
			return model.First(inMapByNameModelProvider(l)(ctx)(mapId, name), model.Filters[Model]())
		}
	}
}

func Enter(l logrus.FieldLogger, ctx context.Context, kp producer.Provider) func(worldId byte, channelId byte, mapId uint32, portalName string, characterId uint32) error {
	return func(worldId byte, channelId byte, mapId uint32, portalName string, characterId uint32) error {
		p, err := GetInMapByName(l)(ctx)(mapId, portalName)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, mapId)
			return err
		}
		err = kp(EnvPortalCommandTopic)(enterCommandProvider(worldId, channelId, mapId, p.id, characterId))
		return err
	}
}
