package mock

import (
	"atlas-npc-conversations/conversation"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorMock is a mock implementation of the conversation.Processor interface
type ProcessorMock struct {
	// StartFunc is a function field for the Start method
	StartFunc func(field field.Model, npcId uint32, characterId uint32) error

	// ContinueFunc is a function field for the Continue method
	ContinueFunc func(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error

	// ContinueViaEventFunc is a function field for the ContinueViaEvent method
	ContinueViaEventFunc func(characterId uint32, action byte, referenceId int32) error

	// EndFunc is a function field for the End method
	EndFunc func(characterId uint32) error

	// ByIdProviderFunc is a function field for the ByIdProvider method
	ByIdProviderFunc func(id uint32) model.Provider[conversation.Model]

	// ByNpcIdProviderFunc is a function field for the ByNpcIdProvider method
	ByNpcIdProviderFunc func(npcId uint32) model.Provider[conversation.Model]

	// AllProviderFunc is a function field for the AllProvider method
	AllProviderFunc func() model.Provider[[]conversation.Model]
}

// Start is a mock implementation of the conversation.Processor.Start method
func (m *ProcessorMock) Start(field field.Model, npcId uint32, characterId uint32) error {
	if m.StartFunc != nil {
		return m.StartFunc(field, npcId, characterId)
	}
	// Default implementation returns nil (success)
	return nil
}

// Continue is a mock implementation of the conversation.Processor.Continue method
func (m *ProcessorMock) Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error {
	if m.ContinueFunc != nil {
		return m.ContinueFunc(npcId, characterId, action, lastMessageType, selection)
	}
	// Default implementation returns nil (success)
	return nil
}

// ContinueViaEvent is a mock implementation of the conversation.Processor.ContinueViaEvent method
func (m *ProcessorMock) ContinueViaEvent(characterId uint32, action byte, referenceId int32) error {
	if m.ContinueViaEventFunc != nil {
		return m.ContinueViaEventFunc(characterId, action, referenceId)
	}
	// Default implementation returns nil (success)
	return nil
}

// End is a mock implementation of the conversation.Processor.End method
func (m *ProcessorMock) End(characterId uint32) error {
	if m.EndFunc != nil {
		return m.EndFunc(characterId)
	}
	// Default implementation returns nil (success)
	return nil
}

// ByIdProvider is a mock implementation of the conversation.Processor.ByIdProvider method
func (m *ProcessorMock) ByIdProvider(id uint32) model.Provider[conversation.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	// Default implementation returns a provider that returns an empty model
	return func() (conversation.Model, error) {
		return conversation.Model{}, nil
	}
}

// ByNpcIdProvider is a mock implementation of the conversation.Processor.ByNpcIdProvider method
func (m *ProcessorMock) ByNpcIdProvider(npcId uint32) model.Provider[conversation.Model] {
	if m.ByNpcIdProviderFunc != nil {
		return m.ByNpcIdProviderFunc(npcId)
	}
	// Default implementation returns a provider that returns an empty model
	return func() (conversation.Model, error) {
		return conversation.Model{}, nil
	}
}

// AllProvider is a mock implementation of the conversation.Processor.AllProvider method
func (m *ProcessorMock) AllProvider() model.Provider[[]conversation.Model] {
	if m.AllProviderFunc != nil {
		return m.AllProviderFunc()
	}
	// Default implementation returns a provider that returns an empty slice
	return func() ([]conversation.Model, error) {
		return []conversation.Model{}, nil
	}
}
