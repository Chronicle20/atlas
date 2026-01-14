package mock

import (
	"atlas-skills/kafka/message"
	"atlas-skills/macro"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/stretchr/testify/mock"
)

// ProcessorMock is a mock implementation of macro.Processor
type ProcessorMock struct {
	mock.Mock
}

// ByCharacterIdProvider mocks the ByCharacterIdProvider method
func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]macro.Model] {
	args := m.Called(characterId)
	return args.Get(0).(model.Provider[[]macro.Model])
}

// Update mocks the Update method
func (m *ProcessorMock) Update(mb *message.Buffer) func(characterId uint32) func(macros []macro.Model) ([]macro.Model, error) {
	args := m.Called(mb)
	return args.Get(0).(func(characterId uint32) func(macros []macro.Model) ([]macro.Model, error))
}

// UpdateAndEmit mocks the UpdateAndEmit method
func (m *ProcessorMock) UpdateAndEmit(characterId uint32, macros []macro.Model) ([]macro.Model, error) {
	args := m.Called(characterId, macros)
	return args.Get(0).([]macro.Model), args.Error(1)
}

// Delete mocks the Delete method
func (m *ProcessorMock) Delete(characterId uint32) error {
	args := m.Called(characterId)
	return args.Error(0)
}
