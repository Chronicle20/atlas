package mock

import "atlas-pets/pet"

type TemporalRegistry struct {
	UpdatePositionFn func(petId uint32, x int16, y int16, fh int16)
	UpdateFn         func(petId uint32, x int16, y int16, stance byte, fh int16)
	UpdateStanceFn   func(petId uint32, stance byte)
	GetByIdFn        func(petId uint32) *pet.TemporalData
	RemoveFn         func(petId uint32)
}

func (m *TemporalRegistry) UpdatePosition(petId uint32, x int16, y int16, fh int16) {
	if m.UpdatePositionFn != nil {
		m.UpdatePositionFn(petId, x, y, fh)
	}
}

func (m *TemporalRegistry) Update(petId uint32, x int16, y int16, stance byte, fh int16) {
	if m.UpdateFn != nil {
		m.UpdateFn(petId, x, y, stance, fh)
	}
}

func (m *TemporalRegistry) UpdateStance(petId uint32, stance byte) {
	if m.UpdateStanceFn != nil {
		m.UpdateStanceFn(petId, stance)
	}
}

func (m *TemporalRegistry) GetById(petId uint32) *pet.TemporalData {
	if m.GetByIdFn != nil {
		return m.GetByIdFn(petId)
	}
	return nil
}

func (m *TemporalRegistry) Remove(petId uint32) {
	if m.RemoveFn != nil {
		m.RemoveFn(petId)
	}
}
