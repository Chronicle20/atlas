package mock

import (
	"atlas-inventory/equipable"

	"github.com/Chronicle20/atlas-model/model"
)

type ProcessorImpl struct {
	ByEquipmentIdModelProviderFn func(equipmentId uint32) model.Provider[equipable.Model]
	GetByIdFn                    func(equipmentId uint32) (equipable.Model, error)
	DeleteFn                     func(equipmentId uint32) error
	CreateFn                     func(itemId uint32) model.Provider[equipable.Model]
}

func (p *ProcessorImpl) ByEquipmentIdModelProvider(equipmentId uint32) model.Provider[equipable.Model] {
	return p.ByEquipmentIdModelProviderFn(equipmentId)
}

func (p *ProcessorImpl) GetById(equipmentId uint32) (equipable.Model, error) {
	return p.GetByIdFn(equipmentId)
}

func (p *ProcessorImpl) Delete(equipmentId uint32) error {
	return p.DeleteFn(equipmentId)
}

func (p *ProcessorImpl) Create(itemId uint32) model.Provider[equipable.Model] {
	return p.CreateFn(itemId)
}
