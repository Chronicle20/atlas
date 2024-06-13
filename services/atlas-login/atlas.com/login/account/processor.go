package account

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strconv"
)

type LoginErr string

func ForAccountByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string, operator model.Operator[Model]) {
	return func(name string, operator model.Operator[Model]) {
		model.IfPresent[Model](ByNameModelProvider(l, span, tenant)(name), operator)
	}
}

func ByNameModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) model.Provider[Model] {
	return func(name string) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestAccountByName(l, span, tenant)(name), makeModel)
	}
}

func ByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestAccountById(l, span, tenant)(id), makeModel)
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

func makeModel(body RestModel) (Model, error) {
	id, err := strconv.ParseUint(body.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}
	m := NewBuilder().
		SetId(uint32(id)).
		SetName(body.Name).
		SetPassword(body.Password).
		SetPin(body.Pin).
		SetPic(body.Pic).
		SetLoggedIn(int(body.LoggedIn)).
		SetLastLogin(body.LastLogin).
		SetGender(body.Gender).
		SetBanned(body.Banned).
		SetTos(body.TOS).
		SetLanguage(body.Language).
		SetCountry(body.Country).
		SetCharacterSlots(body.CharacterSlots).
		Build()
	return m, nil
}
