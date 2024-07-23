package portal

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func inMapByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, id uint32) model.Provider[Model] {
	return func(mapId uint32, id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestInMapById(l, span, tenant)(mapId, id), Extract)
	}
}

func GetInMapById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, id uint32) (Model, error) {
	return func(mapId uint32, id uint32) (Model, error) {
		return inMapByIdModelProvider(l, span, tenant)(mapId, id)()
	}
}
