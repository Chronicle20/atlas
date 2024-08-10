package account

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type LoginErr string

func ByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestAccountById(l, span, tenant)(id), Extract)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return ByIdModelProvider(l, span, tenant)(id)()
	}
}

func IsLoggedIn(_ logrus.FieldLogger, _ opentracing.Span, tenant tenant.Model) func(id uint32) bool {
	return func(id uint32) bool {
		return getRegistry().LoggedIn(Key{Tenant: tenant, Id: id})
	}
}
