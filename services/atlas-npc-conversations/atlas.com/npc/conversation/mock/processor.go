package mock

import (
	"atlas-npc-conversations/conversation"
	"github.com/Chronicle20/atlas-constants/field"
)

// ProcessorMock is a mock implementation of the conversation.Processor interface
type ProcessorMock struct {
	// StartFunc is a function field for the Start method
	StartFunc func(field field.Model, npcId uint32, characterId uint32) error

	// StartQuestFunc is a function field for the StartQuest method
	StartQuestFunc func(field field.Model, questId uint32, npcId uint32, characterId uint32, stateMachine conversation.StateContainer) error

	// ContinueFunc is a function field for the Continue method
	ContinueFunc func(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error

	// EndFunc is a function field for the End method
	EndFunc func(characterId uint32) error
}

// Start is a mock implementation of the conversation.Processor.Start method
func (m *ProcessorMock) Start(field field.Model, npcId uint32, characterId uint32) error {
	if m.StartFunc != nil {
		return m.StartFunc(field, npcId, characterId)
	}
	// Default implementation returns nil (success)
	return nil
}

// StartQuest is a mock implementation of the conversation.Processor.StartQuest method
func (m *ProcessorMock) StartQuest(field field.Model, questId uint32, npcId uint32, characterId uint32, stateMachine conversation.StateContainer) error {
	if m.StartQuestFunc != nil {
		return m.StartQuestFunc(field, questId, npcId, characterId, stateMachine)
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

// End is a mock implementation of the conversation.Processor.End method
func (m *ProcessorMock) End(characterId uint32) error {
	if m.EndFunc != nil {
		return m.EndFunc(characterId)
	}
	// Default implementation returns nil (success)
	return nil
}
