package slots

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
)

func byIdProvider(ctx context.Context) func(equipmentId uint32) model.Provider[Model] {
	return func(equipmentId uint32) model.Provider[Model] {
		return func() (Model, error) {
			return GetEquipmentSlotCache().GetEquipmentSlot(tenant.MustFromContext(ctx), equipmentId)
		}
	}
}

func GetById(ctx context.Context) func(equipmentId uint32) (Model, error) {
	return func(equipmentId uint32) (Model, error) {
		return byIdProvider(ctx)(equipmentId)()
	}
}
