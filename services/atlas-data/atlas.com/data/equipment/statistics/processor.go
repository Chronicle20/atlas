package statistics

import (
	"atlas-data/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func byIdProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) model.Provider[Model] {
	return func(equipmentId uint32) model.Provider[Model] {
		return func() (Model, error) {
			return GetEquipmentCache().GetEquipment(tenant, equipmentId)
		}
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) (Model, error) {
	return func(equipmentId uint32) (Model, error) {
		return byIdProvider(l, span, tenant)(equipmentId)()
	}
}
