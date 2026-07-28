package mock

import (
	"atlas-consumables/pet"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdProviderFunc             func(petId uint64) model.Provider[pet.Model]
	GetByIdFunc                  func(petId uint64) (pet.Model, error)
	ByOwnerProviderFunc          func(ownerId uint32) model.Provider[[]pet.Model]
	GetByOwnerFunc               func(ownerId uint32) ([]pet.Model, error)
	SpawnedByOwnerProviderFunc   func(ownerId uint32) model.Provider[[]pet.Model]
	HungryByOwnerProviderFunc    func(ownerId uint32) model.Provider[[]pet.Model]
	HungriestByOwnerProviderFunc func(ownerId uint32) model.Provider[pet.Model]
	AwardFullnessFunc            func(actorId uint32, petId uint64, amount byte) error
}

var _ pet.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(petId uint64) model.Provider[pet.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(petId)
	}
	return model.FixedProvider(pet.Model{})
}

func (m *ProcessorMock) GetById(petId uint64) (pet.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(petId)
	}
	return pet.Model{}, nil
}

func (m *ProcessorMock) ByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.ByOwnerProviderFunc != nil {
		return m.ByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) GetByOwner(ownerId uint32) ([]pet.Model, error) {
	if m.GetByOwnerFunc != nil {
		return m.GetByOwnerFunc(ownerId)
	}
	return nil, nil
}

func (m *ProcessorMock) SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.SpawnedByOwnerProviderFunc != nil {
		return m.SpawnedByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) HungryByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.HungryByOwnerProviderFunc != nil {
		return m.HungryByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) HungriestByOwnerProvider(ownerId uint32) model.Provider[pet.Model] {
	if m.HungriestByOwnerProviderFunc != nil {
		return m.HungriestByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider(pet.Model{})
}

func (m *ProcessorMock) AwardFullness(actorId uint32, petId uint64, amount byte) error {
	if m.AwardFullnessFunc != nil {
		return m.AwardFullnessFunc(actorId, petId, amount)
	}
	return nil
}
