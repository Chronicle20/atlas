package cash

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byEquipmentIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(itemId uint32) model.Provider[Model] {
		return func(itemId uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(itemId), Extract)
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) (Model, error) {
	return func(ctx context.Context) func(itemId uint32) (Model, error) {
		return func(itemId uint32) (Model, error) {
			return byEquipmentIdModelProvider(l)(ctx)(itemId)()
		}
	}
}
