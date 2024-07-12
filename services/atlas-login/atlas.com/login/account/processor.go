package account

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type LoginErr string

func ForAccountByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string, operator model.Operator[Model]) {
	return func(name string, operator model.Operator[Model]) {
		model.IfPresent[Model](ByNameModelProvider(l, span, tenant)(name), operator)
	}
}

func ByNameModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) model.Provider[Model] {
	return func(name string) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestAccountByName(l, span, tenant)(name), Extract)
	}
}

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

func IsLoggedIn(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) bool {
	return func(id uint32) bool {
		a, err := GetById(l, span, tenant)(id)
		if err != nil {
			return false
		} else if a.LoggedIn() != 0 {
			return true
		} else {
			return false
		}
	}
}

func UpdatePin(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32, pin string) error {
	return func(id uint32, pin string) error {
		a, err := GetById(l, span, tenant)(id)
		if err != nil {
			return err
		}
		a.pin = pin
		_, err = requestUpdate(l, span, tenant)(a)(l)
		if err != nil {
			return err
		}
		return nil
	}
}
