package portal

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func inMapByNameModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, name string) model.Provider[[]Model] {
	return func(mapId uint32, name string) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestInMapByName(ctx, tenant)(mapId, name), Extract)
	}
}

func GetInMapByName(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, name string) (Model, error) {
	return func(mapId uint32, name string) (Model, error) {
		return model.First(inMapByNameModelProvider(l, ctx, tenant)(mapId, name))
	}
}

func Enter(l logrus.FieldLogger, ctx context.Context, kp producer.Provider) func(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, portalName string, characterId uint32) error {
	return func(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, portalName string, characterId uint32) error {
		p, err := GetInMapByName(l, ctx, tenant)(mapId, portalName)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, mapId)
			return err
		}
		err = kp(EnvPortalCommandTopic)(enterCommandProvider(tenant)(worldId, channelId, mapId, p.id, characterId))
		return err
	}
}
