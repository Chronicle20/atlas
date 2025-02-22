package portal

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func inMapByNameModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, name string) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId _map.Id, name string) model.Provider[[]Model] {
		return func(mapId _map.Id, name string) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
		}
	}
}

func GetInMapByName(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, name string) (Model, error) {
	return func(ctx context.Context) func(mapId _map.Id, name string) (Model, error) {
		return func(mapId _map.Id, name string) (Model, error) {
			return model.First(inMapByNameModelProvider(l)(ctx)(mapId, name), model.Filters[Model]())
		}
	}
}

func Enter(l logrus.FieldLogger, ctx context.Context, kp producer.Provider) func(m _map.Model, portalName string, characterId uint32) error {
	return func(m _map.Model, portalName string, characterId uint32) error {
		p, err := GetInMapByName(l)(ctx)(m.MapId(), portalName)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, m.MapId())
			return err
		}
		err = kp(EnvPortalCommandTopic)(enterCommandProvider(m, p.id, characterId))
		return err
	}
}
