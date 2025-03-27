package statistics

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Creator func(itemId uint32) model.Provider[Model]

func Create(l logrus.FieldLogger) func(ctx context.Context) Creator {
	return func(ctx context.Context) Creator {
		return func(itemId uint32) model.Provider[Model] {
			ro, err := requestCreate(itemId)(l, ctx)
			if err != nil {
				l.WithError(err).Errorf("Generating equipment item %d, they were not awarded this item. Check request in ESO service.", itemId)
				return model.ErrorProvider[Model](err)
			}
			return model.Map(Extract)(model.FixedProvider(ro))
		}
	}
}

func Existing(l logrus.FieldLogger) func(ctx context.Context) func(equipmentId uint32) Creator {
	return func(ctx context.Context) func(equipmentId uint32) Creator {
		return func(equipmentId uint32) Creator {
			return func(itemId uint32) model.Provider[Model] {
				return byEquipmentIdModelProvider(l, ctx)(equipmentId)
			}
		}
	}
}

func byEquipmentIdModelProvider(l logrus.FieldLogger, ctx context.Context) func(equipmentId uint32) model.Provider[Model] {
	return func(equipmentId uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l, ctx)(requestById(equipmentId), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context) func(equipmentId uint32) (Model, error) {
	return func(equipmentId uint32) (Model, error) {
		return byEquipmentIdModelProvider(l, ctx)(equipmentId)()
	}
}

func UpdateById(l logrus.FieldLogger, ctx context.Context) func(equipmentId uint32, i RestModel) (Model, error) {
	return func(equipmentId uint32, i RestModel) (Model, error) {
		orm, err := updateById(equipmentId, i)(l, ctx)
		if err != nil {
			return Model{}, err
		}
		return Extract(orm)
	}
}

func Delete(l logrus.FieldLogger, ctx context.Context) func(equipmentId uint32) error {
	return func(equipmentId uint32) error {
		return deleteById(equipmentId)(l, ctx)
	}
}
