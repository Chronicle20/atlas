package _map

import (
	"atlas-messages/character"
	"atlas-messages/kafka/producer"
	"atlas-messages/portal"
	"atlas-messages/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func Exists(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32) bool {
	return func(mapId uint32) bool {
		_, err := requests.Provider[RestModel, Model](l)(requestMap(ctx, tenant)(mapId), Extract)()
		if err != nil {
			l.WithError(err).Errorf("Unable to find requested map [%d].", mapId)
			return false
		}
		return true
	}
}

func WarpRandom(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32, mapId uint32) error {
	return func(worldId byte, channelId byte, characterId uint32, mapId uint32) error {
		return WarpToPortal(l, ctx, tenant)(worldId, channelId, characterId, mapId, portal.RandomPortalIdProvider(l, ctx, tenant)(mapId))
	}
}

func WarpToPortal(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
	return func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
		id, err := p()
		if err != nil {
			return err
		}

		return producer.ProviderImpl(l)(ctx)(character.EnvCommandTopic)(character.ChangeMapProvider(tenant, worldId, channelId, characterId, mapId, id))
	}
}
