package account

import (
	"atlas-login/rest/requests"
	"atlas-login/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
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
		return requests.Provider[RestModel, Model](l, span, tenant)(requestAccountByName(name), makeModel)
	}
}

func AttemptLogin(l logrus.FieldLogger, span opentracing.Span) func(t tenant.Model, sessionId uuid.UUID, name string, password string) ([]LoginErr, error) {
	return func(t tenant.Model, sessionId uuid.UUID, name string, password string) ([]LoginErr, error) {
		return nil, errors.New("not implemented")
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
