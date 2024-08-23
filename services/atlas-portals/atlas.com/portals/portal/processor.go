package portal

import (
	"atlas-portals/character"
	"atlas-portals/kafka/producer"
	"atlas-portals/tenant"
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

func inMapByIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, id uint32) model.Provider[Model] {
	return func(mapId uint32, id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestInMapById(ctx, tenant)(mapId, id), Extract)
	}
}

func GetInMapByName(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, name string) (Model, error) {
	return func(mapId uint32, name string) (Model, error) {
		return model.First(inMapByNameModelProvider(l, ctx, tenant)(mapId, name))
	}
}

func GetInMapById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, id uint32) (Model, error) {
	return func(mapId uint32, id uint32) (Model, error) {
		return inMapByIdModelProvider(l, ctx, tenant)(mapId, id)()
	}
}

func Enter(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
		l.Debugf("Character [%d] entering portal [%d] in map [%d].", characterId, portalId, mapId)
		p, err := GetInMapById(l, ctx, tenant)(mapId, portalId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate portal [%d] in map [%d] character [%d] is trying to enter.", portalId, mapId, characterId)
			return
		}

		if p.HasScript() {
			l.Debugf("Portal [%s] has script. Executing [%s] for character [%d].", p.String(), p.ScriptName(), characterId)
			character.EnableActions(l, ctx, tenant)(worldId, channelId, characterId)
			return
		}

		if p.HasTargetMap() {
			l.Debugf("Portal [%s] has target. Transfering character [%d] to [%d].", p.String(), characterId, p.TargetMapId())

			var tp Model
			tp, err = GetInMapByName(l, ctx, tenant)(p.TargetMapId(), p.Target())
			if err != nil {
				l.WithError(err).Warnf("Unable to locate portal target [%s] for map [%d]. Defaulting to portal 0.", p.Target(), p.TargetMapId())
				tp, err = GetInMapById(l, ctx, tenant)(p.TargetMapId(), 0)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate portal 0 for map [%d]. Is there invalid wz data?", p.TargetMapId())
					character.EnableActions(l, ctx, tenant)(worldId, channelId, characterId)
					return
				}
			}
			WarpById(l, ctx, tenant)(worldId, channelId, characterId, p.TargetMapId(), tp.Id())
			return
		}

		character.EnableActions(l, ctx, tenant)(worldId, channelId, characterId)
	}
}

func WarpById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
	return func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
		WarpToPortal(l, ctx, tenant)(worldId, channelId, characterId, mapId, model.FixedProvider(portalId))
	}
}

func WarpToPortal(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
	return func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
		id, err := p()
		if err == nil {
			_ = producer.ProviderImpl(l)(ctx)(character.EnvCommandTopic)(character.ChangeMapProvider(tenant, worldId, channelId, characterId, mapId, id))
		}
	}
}
