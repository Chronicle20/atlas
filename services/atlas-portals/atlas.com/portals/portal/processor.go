package portal

import (
	"atlas-portals/character"
	"atlas-portals/kafka/producer"
	"atlas-portals/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func inMapByNameModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, name string) model.Provider[[]Model] {
	return func(mapId uint32, name string) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestInMapByName(l, span, tenant)(mapId, name), Extract)
	}
}

func inMapByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, id uint32) model.Provider[Model] {
	return func(mapId uint32, id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestInMapById(l, span, tenant)(mapId, id), Extract)
	}
}

func GetInMapByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, name string) (Model, error) {
	return func(mapId uint32, name string) (Model, error) {
		return model.First(inMapByNameModelProvider(l, span, tenant)(mapId, name))
	}
}

func GetInMapById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, id uint32) (Model, error) {
	return func(mapId uint32, id uint32) (Model, error) {
		return inMapByIdModelProvider(l, span, tenant)(mapId, id)()
	}
}

func Enter(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) {
		l.Debugf("Character [%d] entering portal [%d] in map [%d].", characterId, portalId, mapId)
		p, err := GetInMapById(l, span, tenant)(mapId, portalId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate portal [%d] in map [%d] character [%d] is trying to enter.", portalId, mapId, characterId)
			return
		}

		if p.HasScript() {
			l.Debugf("Portal [%s] has script. Executing [%s] for character [%d].", p.String(), p.ScriptName(), characterId)
			character.EnableActions(l, span, tenant)(worldId, channelId, characterId)
			return
		}

		if p.HasTargetMap() {
			l.Debugf("Portal [%s] has target. Transfering character [%d] to [%d].", p.String(), characterId, p.TargetMapId())

			var tp Model
			tp, err = GetInMapByName(l, span, tenant)(p.TargetMapId(), p.Target())
			if err != nil {
				l.WithError(err).Warnf("Unable to locate portal target [%s] for map [%d]. Defaulting to portal 0.", p.Target(), p.TargetMapId())
				tp, err = GetInMapById(l, span, tenant)(p.TargetMapId(), 0)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate portal 0 for map [%d]. Is there invalid wz data?", p.TargetMapId())
					character.EnableActions(l, span, tenant)(worldId, channelId, characterId)
					return
				}
			}
			WarpById(l, span, tenant)(worldId, channelId, characterId, p.TargetMapId(), tp.Id())
			return
		}

		character.EnableActions(l, span, tenant)(worldId, channelId, characterId)
	}
}

func WarpById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
	return func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) {
		WarpToPortal(l, span, tenant)(worldId, channelId, characterId, mapId, model.FixedProvider(portalId))
	}
}

func WarpToPortal(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
	return func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) {
		id, err := p()
		if err == nil {
			_ = producer.ProviderImpl(l)(span)(character.EnvCommandTopic)(character.ChangeMapProvider(tenant, worldId, channelId, characterId, mapId, id))
		}
	}
}
