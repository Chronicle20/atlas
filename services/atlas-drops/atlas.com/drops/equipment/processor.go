package equipment

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
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

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(equipmentId uint32) error {
	return func(ctx context.Context) func(equipmentId uint32) error {
		return func(equipmentId uint32) error {
			return deleteById(equipmentId)(l, ctx)
		}
	}
}
