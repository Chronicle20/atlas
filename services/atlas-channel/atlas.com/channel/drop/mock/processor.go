package mock

import (
	"atlas-channel/drop"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InMapModelProviderFunc func(f field.Model) model.Provider[[]drop.Model]
	ForEachInMapFunc       func(f field.Model, o model.Operator[drop.Model]) error
	RequestReservationFunc func(f field.Model, dropId uint32, characterId uint32, partyId uint32, characterX int16, characterY int16, petSlot int8) error
}

var _ drop.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InMapModelProvider(f field.Model) model.Provider[[]drop.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(f)
	}
	return model.FixedProvider([]drop.Model{})
}

func (m *ProcessorMock) ForEachInMap(f field.Model, o model.Operator[drop.Model]) error {
	if m.ForEachInMapFunc != nil {
		return m.ForEachInMapFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) RequestReservation(f field.Model, dropId uint32, characterId uint32, partyId uint32, characterX int16, characterY int16, petSlot int8) error {
	if m.RequestReservationFunc != nil {
		return m.RequestReservationFunc(f, dropId, characterId, partyId, characterX, characterY, petSlot)
	}
	return nil
}
