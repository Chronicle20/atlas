package information

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByIdModelProvider(l logrus.FieldLogger, ctx context.Context) func(id uint32) model.Provider[[]Model] {
	return func(id uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestEquipmentSlotDestination(id), Extract, model.Filters[Model]())
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context) func(id uint32) ([]Model, error) {
	return func(id uint32) ([]Model, error) {
		return ByIdModelProvider(l, ctx)(id)()
	}
}
