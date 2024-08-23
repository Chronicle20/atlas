package statistics

import (
	"atlas-character/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Creator func(itemId uint32) model.Provider[Model]

func Create(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) Creator {
	return func(itemId uint32) model.Provider[Model] {
		ro, err := requestCreate(ctx, tenant)(itemId)(l)
		if err != nil {
			l.WithError(err).Errorf("Generating equipment item %d, they were not awarded this item. Check request in ESO service.", itemId)
			return model.ErrorProvider[Model](err)
		}
		return model.Map(model.FixedProvider(ro), makeEquipment)
	}
}

func byEquipmentIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(equipmentId uint32) model.Provider[Model] {
	return func(equipmentId uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestById(ctx, tenant)(equipmentId), makeEquipment)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(equipmentId uint32) (Model, error) {
	return func(equipmentId uint32) (Model, error) {
		return byEquipmentIdModelProvider(l, ctx, tenant)(equipmentId)()
	}
}

func Delete(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(equipmentId uint32) error {
	return func(equipmentId uint32) error {
		return deleteById(ctx, tenant)(equipmentId)(l)
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
