package equipable

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byEquipmentIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(equipmentId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(equipmentId uint32) model.Provider[Model] {
		return func(equipmentId uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(equipmentId), Extract)
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(equipmentId uint32) (Model, error) {
	return func(ctx context.Context) func(equipmentId uint32) (Model, error) {
		return func(equipmentId uint32) (Model, error) {
			return byEquipmentIdModelProvider(l)(ctx)(equipmentId)()
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(equipmentId uint32) error {
	return func(ctx context.Context) func(equipmentId uint32) error {
		return func(equipmentId uint32) error {
			return deleteById(equipmentId)(l, ctx)
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(itemId uint32) model.Provider[Model] {
		return func(itemId uint32) model.Provider[Model] {
			ro, err := requestCreate(itemId)(l, ctx)
			if err != nil {
				l.WithError(err).Errorf("Unable to generate equipable information.")
				return model.ErrorProvider[Model](err)
			}
			return model.Map(Extract)(model.FixedProvider(ro))
		}
	}
}
