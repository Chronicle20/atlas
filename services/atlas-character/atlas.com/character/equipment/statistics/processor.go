package statistics

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func ByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestById(l, span, tenant)(id), Extract)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return ByIdModelProvider(l, span, tenant)(id)()
	}
}
