package npc

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Model represents a conversation tree for an NPC
type Model struct {
	id         uuid.UUID
	npcId      uint32
	startState string
	states     []conversation.StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

// GetId returns the conversation ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// GetNpcId returns the NPC ID
func (m Model) NpcId() uint32 {
	return m.npcId
}

// GetStartState returns the starting state ID
func (m Model) StartState() string {
	return m.startState
}

// GetStates returns the conversation states
func (m Model) States() []conversation.StateModel {
	return m.states
}

// GetCreatedAt returns the creation timestamp
func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

// GetUpdatedAt returns the last update timestamp
func (m Model) UpdatedAt() time.Time {
	return m.updatedAt
}

// FindState finds a state by ID - implements StateContainer interface
func (m Model) FindState(stateId string) (conversation.StateModel, error) {
	for _, state := range m.states {
		if state.Id() == stateId {
			return state, nil
		}
	}
	return conversation.StateModel{}, errors.New("state not found")
}
