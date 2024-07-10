package statistics

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(itemId uint32) (uint32, error) {
	return func(itemId uint32) (uint32, error) {
		ro, err := requestCreate(l, span, tenant)(itemId)(l)
		if err != nil {
			l.WithError(err).Errorf("Generating equipment item %d, they were not awarded this item. Check request in ESO service.", itemId)
			return 0, err
		}
		return ro.Id, nil
	}
}

func byEquipmentIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) model.Provider[Model] {
	return func(equipmentId uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestById(l, span, tenant)(equipmentId), makeEquipment)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) (Model, error) {
	return func(equipmentId uint32) (Model, error) {
		return byEquipmentIdModelProvider(l, span, tenant)(equipmentId)()
	}
}

func Delete(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) error {
	return func(equipmentId uint32) error {
		return deleteById(l, span, tenant)(equipmentId)(l)
	}
}

func makeEquipment(resp RestModel) (Model, error) {
	return Model{
		id:            resp.Id,
		itemId:        resp.ItemId,
		strength:      resp.Strength,
		dexterity:     resp.Dexterity,
		intelligence:  resp.Intelligence,
		luck:          resp.Luck,
		hp:            resp.HP,
		mp:            resp.MP,
		weaponAttack:  resp.WeaponAttack,
		magicAttack:   resp.MagicAttack,
		weaponDefense: resp.WeaponDefense,
		magicDefense:  resp.MagicDefense,
		accuracy:      resp.Accuracy,
		avoidability:  resp.Avoidability,
		hands:         resp.Hands,
		speed:         resp.Speed,
		jump:          resp.Jump,
		slots:         resp.Slots,
	}, nil
}
