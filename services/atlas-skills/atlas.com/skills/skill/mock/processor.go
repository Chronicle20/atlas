package mock

import (
	"atlas-skills/kafka/message"
	"atlas-skills/skill"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/stretchr/testify/mock"
)

// ProcessorMock is a mock implementation of skill.Processor
type ProcessorMock struct {
	mock.Mock
}

// ByCharacterIdProvider mocks the ByCharacterIdProvider method
func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]skill.Model] {
	args := m.Called(characterId)
	return args.Get(0).(model.Provider[[]skill.Model])
}

// ByIdProvider mocks the ByIdProvider method
func (m *ProcessorMock) ByIdProvider(characterId uint32, id uint32) model.Provider[skill.Model] {
	args := m.Called(characterId, id)
	return args.Get(0).(model.Provider[skill.Model])
}

// Create mocks the Create method
func (m *ProcessorMock) Create(mb *message.Buffer) func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (skill.Model, error) {
	args := m.Called(mb)
	return args.Get(0).(func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (skill.Model, error))
}

// CreateAndEmit mocks the CreateAndEmit method
func (m *ProcessorMock) CreateAndEmit(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (skill.Model, error) {
	args := m.Called(characterId, id, level, masterLevel, expiration)
	return args.Get(0).(skill.Model), args.Error(1)
}

// Update mocks the Update method
func (m *ProcessorMock) Update(mb *message.Buffer) func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (skill.Model, error) {
	args := m.Called(mb)
	return args.Get(0).(func(characterId uint32) func(id uint32) func(level byte) func(masterLevel byte) func(expiration time.Time) (skill.Model, error))
}

// UpdateAndEmit mocks the UpdateAndEmit method
func (m *ProcessorMock) UpdateAndEmit(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (skill.Model, error) {
	args := m.Called(characterId, id, level, masterLevel, expiration)
	return args.Get(0).(skill.Model), args.Error(1)
}

// SetCooldown mocks the SetCooldown method
func (m *ProcessorMock) SetCooldown(mb *message.Buffer) func(characterId uint32) func(skillId uint32) func(cooldown uint32) (skill.Model, error) {
	args := m.Called(mb)
	return args.Get(0).(func(characterId uint32) func(skillId uint32) func(cooldown uint32) (skill.Model, error))
}

// SetCooldownAndEmit mocks the SetCooldownAndEmit method
func (m *ProcessorMock) SetCooldownAndEmit(characterId uint32, skillId uint32, cooldown uint32) (skill.Model, error) {
	args := m.Called(characterId, skillId, cooldown)
	return args.Get(0).(skill.Model), args.Error(1)
}

// ClearAll mocks the ClearAll method
func (m *ProcessorMock) ClearAll(characterId uint32) error {
	args := m.Called(characterId)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *ProcessorMock) Delete(characterId uint32) error {
	args := m.Called(characterId)
	return args.Error(0)
}

// CooldownDecorator mocks the CooldownDecorator method
func (m *ProcessorMock) CooldownDecorator(characterId uint32) model.Decorator[skill.Model] {
	args := m.Called(characterId)
	return args.Get(0).(model.Decorator[skill.Model])
}

// RequestCreate mocks the RequestCreate method
func (m *ProcessorMock) RequestCreate(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	args := m.Called(characterId, id, level, masterLevel, expiration)
	return args.Error(0)
}

// RequestUpdate mocks the RequestUpdate method
func (m *ProcessorMock) RequestUpdate(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	args := m.Called(characterId, id, level, masterLevel, expiration)
	return args.Error(0)
}
