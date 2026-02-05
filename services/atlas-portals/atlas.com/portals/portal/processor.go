package portal

import (
	"atlas-portals/blocked"
	"atlas-portals/character"
	"atlas-portals/kafka/producer"
	"atlas-portals/portal_actions"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InMapByNameProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, name string) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId _map.Id, name string) model.Provider[[]Model] {
		return func(mapId _map.Id, name string) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
		}
	}
}

func InMapByIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, id uint32) model.Provider[Model] {
	return func(ctx context.Context) func(mapId _map.Id, id uint32) model.Provider[Model] {
		return func(mapId _map.Id, id uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestInMapById(mapId, id), Extract)
		}
	}
}

func GetInMapByName(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, name string) (Model, error) {
	return func(ctx context.Context) func(mapId _map.Id, name string) (Model, error) {
		return func(mapId _map.Id, name string) (Model, error) {
			return model.First(InMapByNameProvider(l)(ctx)(mapId, name), model.Filters[Model]())
		}
	}
}

func GetInMapById(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id, id uint32) (Model, error) {
	return func(ctx context.Context) func(mapId _map.Id, id uint32) (Model, error) {
		return func(mapId _map.Id, id uint32) (Model, error) {
			return InMapByIdProvider(l)(ctx)(mapId, id)()
		}
	}
}

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, portalId uint32, characterId uint32) {
	return func(ctx context.Context) func(f field.Model, portalId uint32, characterId uint32) {
		return func(f field.Model, portalId uint32, characterId uint32) {
			l.Debugf("Character [%d] entering portal [%d] in map [%d].", characterId, portalId, f.MapId())

			// Check if the portal is blocked for this character
			t := tenant.MustFromContext(ctx)
			if blocked.GetCache().IsBlocked(t.Id(), characterId, f.MapId(), portalId) {
				l.Debugf("Portal [%d] in map [%d] is blocked for character [%d]. Enabling actions and returning.", portalId, f.MapId(), characterId)
				character.EnableActions(l)(ctx)(f, characterId)
				return
			}

			p, err := GetInMapById(l)(ctx)(f.MapId(), portalId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate portal [%d] in map [%d] character [%d] is trying to enter.", portalId, f.MapId(), characterId)
				return
			}

			if p.HasScript() {
				l.Debugf("Portal [%s] has script. Executing [%s] for character [%d].", p.String(), p.ScriptName(), characterId)
				portal_actions.ExecuteScript(l)(ctx)(f, portalId, characterId, p.ScriptName())
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
						character.EnableActions(l)(ctx)(f, characterId)
						return
					}
				}
				WarpById(l)(ctx)(f, characterId, p.TargetMapId(), tp.Id())
				return
			}

			character.EnableActions(l)(ctx)(f, characterId)
		}
	}
}

func WarpById(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) {
	return func(ctx context.Context) func(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) {
		return func(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) {
			WarpToPortal(l)(ctx)(f, characterId, targetMapId, model.FixedProvider(portalId))
		}
	}
}

func WarpToPortal(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32, targetMapId _map.Id, p model.Provider[uint32]) {
	return func(ctx context.Context) func(f field.Model, characterId uint32, targetMapId _map.Id, p model.Provider[uint32]) {
		return func(f field.Model, characterId uint32, targetMapId _map.Id, p model.Provider[uint32]) {
			id, err := p()
			if err == nil {
				_ = producer.ProviderImpl(l)(ctx)(character.EnvCommandTopic)(character.ChangeMapProvider(f, characterId, targetMapId, id))
			}
		}
	}
}
