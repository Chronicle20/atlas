package portal

import (
	"atlas-portals/character"
	"atlas-portals/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func InMapByNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, name string) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32, name string) model.Provider[[]Model] {
		return func(mapId uint32, name string) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
		}
	}
}

func InMapByIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, id uint32) model.Provider[Model] {
	return func(ctx context.Context) func(mapId uint32, id uint32) model.Provider[Model] {
		return func(mapId uint32, id uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestInMapById(mapId, id), Extract)
		}
	}
}

func GetInMapByName(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, name string) (Model, error) {
	return func(ctx context.Context) func(mapId uint32, name string) (Model, error) {
		return func(mapId uint32, name string) (Model, error) {
			return model.First(InMapByNameProvider(l)(ctx)(mapId, name), model.Filters[Model]())
		}
	}
}

func GetInMapById(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, id uint32) (Model, error) {
	return func(ctx context.Context) func(mapId uint32, id uint32) (Model, error) {
		return func(mapId uint32, id uint32) (Model, error) {
			return InMapByIdProvider(l)(ctx)(mapId, id)()
		}
	}
}

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
		return func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
			l.Debugf("Character [%d] entering portal [%d] in map [%d].", characterId, portalId, mapId)
			p, err := GetInMapById(l)(ctx)(mapId, portalId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate portal [%d] in map [%d] character [%d] is trying to enter.", portalId, mapId, characterId)
				return
			}

			if p.HasScript() {
				l.Debugf("Portal [%s] has script. Executing [%s] for character [%d].", p.String(), p.ScriptName(), characterId)
				character.EnableActions(l)(ctx)(worldId, channelId, characterId)
				return
			}

			if p.HasTargetMap() {
				l.Debugf("Portal [%s] has target. Transfering character [%d] to [%d].", p.String(), characterId, p.TargetMapId())

				var tp Model
				tp, err = GetInMapByName(l)(ctx)(p.TargetMapId(), p.Target())
				if err != nil {
					l.WithError(err).Warnf("Unable to locate portal target [%s] for map [%d]. Defaulting to portal 0.", p.Target(), p.TargetMapId())
					tp, err = GetInMapById(l)(ctx)(p.TargetMapId(), 0)
					if err != nil {
						l.WithError(err).Errorf("Unable to locate portal 0 for map [%d]. Is there invalid wz data?", p.TargetMapId())
						character.EnableActions(l)(ctx)(worldId, channelId, characterId)
						return
					}
				}
				WarpById(l)(ctx)(worldId, channelId, characterId, p.TargetMapId(), tp.Id())
				return
			}

			character.EnableActions(l)(ctx)(worldId, channelId, characterId)
		}
	}
}

func WarpById(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
			WarpToPortal(l)(ctx)(worldId, channelId, characterId, mapId, model.FixedProvider(portalId))
		}
	}
}

func WarpToPortal(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
			id, err := p()
			if err == nil {
				_ = producer.ProviderImpl(l)(ctx)(character.EnvCommandTopic)(character.ChangeMapProvider(worldId, channelId, characterId, mapId, id))
			}
		}
	}
}
