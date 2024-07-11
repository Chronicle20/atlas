package data

import (
	"atlas-character-factory/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func byIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) model.SliceProvider[Model] {
	return func(id uint32) model.SliceProvider[Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestById(l, span, tenant)(id), Extract)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) ([]Model, error) {
	return func(id uint32) ([]Model, error) {
		return byIdModelProvider(l, span, tenant)(id)()
	}
}
