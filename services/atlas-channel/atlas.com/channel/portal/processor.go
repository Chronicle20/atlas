package portal

import (
	"atlas-channel/tenant"
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

func GetInMapByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, name string) (Model, error) {
	return func(mapId uint32, name string) (Model, error) {
		return model.First(inMapByNameModelProvider(l, span, tenant)(mapId, name))
	}
}

func Enter(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalName string, characterId uint32) error {
	return func(worldId byte, channelId byte, mapId uint32, portalName string, characterId uint32) error {
		p, err := GetInMapByName(l, span, tenant)(mapId, portalName)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, mapId)
			return err
		}
		emitEnterCommand(l, span, tenant)(worldId, channelId, mapId, p.id, characterId)
		return nil
	}
}
