package properties

import (
	"atlas-login/rest/requests"
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strconv"
)

func GetByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		return requests.Provider[RestModel, Model](l, span, tenant)(requestPropertiesByName(name), MakeModel)()
	}
}

func ForWorldProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte) model.SliceProvider[Model] {
	return func(accountId uint32, worldId byte) model.SliceProvider[Model] {
		return requests.SliceProvider[RestModel, Model](l, span, tenant)(requestPropertiesByAccountAndWorld(accountId, worldId), MakeModel)
	}
}

func GetForWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte) ([]Model, error) {
	return func(accountId uint32, worldId byte) ([]Model, error) {
		return ForWorldProvider(l, span, tenant)(accountId, worldId)()
	}
}

func ByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l, span, tenant)(requestPropertiesById(id), MakeModel)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return ByIdModelProvider(l, span, tenant)(characterId)()
	}
}

func MakeModel(ca RestModel) (Model, error) {
	cid, err := strconv.ParseUint(ca.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}
	r := NewBuilder().
		SetId(uint32(cid)).
		SetWorldId(ca.WorldId).
		SetName(ca.Name).
		SetGender(ca.Gender).
		SetSkinColor(ca.SkinColor).
		SetFace(ca.Face).
		SetHair(ca.Hair).
		SetLevel(ca.Level).
		SetJobId(ca.JobId).
		SetStrength(ca.Strength).
		SetDexterity(ca.Dexterity).
		SetIntelligence(ca.Intelligence).
		SetLuck(ca.Luck).
		SetHp(ca.Hp).
		SetMaxHp(ca.MaxHp).
		SetMp(ca.Mp).
		SetMaxMp(ca.MaxMp).
		SetAp(ca.Ap).
		SetSp(ca.Sp).
		SetExperience(ca.Experience).
		SetFame(ca.Fame).
		SetGachaponExperience(ca.GachaponExperience).
		SetMapId(ca.MapId).
		SetSpawnPoint(ca.SpawnPoint).
		Build()
	return r, nil
}
