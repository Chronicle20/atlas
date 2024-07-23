package portal

import (
	"atlas-portals/character"
	"atlas-portals/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func inMapByNameModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, name string) model.SliceProvider[Model] {
	return func(mapId uint32, name string) model.SliceProvider[Model] {
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
			character.EnableActions(l, span, tenant)(worldId, channelId, characterId)
			return
		}

		character.EnableActions(l, span, tenant)(worldId, channelId, characterId)
	}
}
