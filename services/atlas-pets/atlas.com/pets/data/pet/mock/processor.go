package mock

import "atlas-pets/data/pet"

type Processor struct {
	GetByIdFn func(petId uint32) (pet.Model, error)
}

func (m *Processor) GetById(petId uint32) (pet.Model, error) {
	if m.GetByIdFn != nil {
		return m.GetByIdFn(petId)
	}
	return pet.Model{}, nil
}
