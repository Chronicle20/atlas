package mock

import (
	"atlas-inventory/data/consumable"
)

type ProcessorImpl struct {
	GetByIdFn         func(itemId uint32) (consumable.Model, error)
	GetRechargeableFn func() ([]consumable.Model, error)
}

func (p *ProcessorImpl) GetById(itemId uint32) (consumable.Model, error) {
	return p.GetByIdFn(itemId)
}

func (p *ProcessorImpl) GetRechargeable() ([]consumable.Model, error) {
	return p.GetRechargeableFn()
}
