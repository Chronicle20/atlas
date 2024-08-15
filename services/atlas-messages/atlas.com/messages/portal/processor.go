package portal

import (
	"atlas-messages/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func InMapProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) model.Provider[[]Model] {
	return func(mapId uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestAll(l, span, tenant)(mapId), Extract)
	}
}

func RandomPortalProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) model.Provider[Model] {
	return func(mapId uint32) model.Provider[Model] {
		return func() (Model, error) {
			ps, err := InMapProvider(l, span, tenant)(mapId)()
			if err != nil {
				return Model{}, err
			}
			return model.RandomPreciselyOneFilter(ps)
		}
	}
}

func RandomPortalIdProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) model.Provider[uint32] {
	return func(mapId uint32) model.Provider[uint32] {
		return model.Map(RandomPortalProvider(l, span, tenant)(mapId), getId)
	}
}

func getId(m Model) (uint32, error) {
	return m.Id(), nil
}
