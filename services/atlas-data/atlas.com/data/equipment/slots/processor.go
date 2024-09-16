package slots

import (
	"atlas-data/tenant"
	"github.com/Chronicle20/atlas-model/model"
)

func byIdProvider(tenant tenant.Model) func(equipmentId uint32) model.Provider[Model] {
	return func(equipmentId uint32) model.Provider[Model] {
		return func() (Model, error) {
			return GetEquipmentSlotCache().GetEquipmentSlot(tenant, equipmentId)
		}
	}
}

func GetById(tenant tenant.Model) func(equipmentId uint32) (Model, error) {
	return func(equipmentId uint32) (Model, error) {
		return byIdProvider(tenant)(equipmentId)()
	}
}
