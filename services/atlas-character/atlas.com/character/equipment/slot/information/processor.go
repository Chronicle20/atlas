package information

import (
	"atlas-character/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) model.Provider[[]Model] {
	return func(id uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestEquipmentSlotDestination(ctx, tenant)(id), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) ([]Model, error) {
	return func(id uint32) ([]Model, error) {
		return ByIdModelProvider(l, ctx, tenant)(id)()
	}
}
