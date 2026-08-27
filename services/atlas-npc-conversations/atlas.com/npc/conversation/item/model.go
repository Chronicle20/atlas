package item

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Model represents a conversation attached to a scripted item (the 243xxxx
// family). Resolution is by item id; npcId names only the avatar the dialogue
// renders with, and scriptName records the WZ spec/script value for authoring
// traceability — neither is a lookup key.
//
// The shape mirrors conversation/npc (a single state machine) rather than
// conversation/quest (a start/end pair): a scripted item has exactly one
// entry point, so a second machine would be permanently nil.
type Model struct {
	id         uuid.UUID
	itemId     uint32
	npcId      uint32
	scriptName string
	startState string
	states     []conversation.StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

// Id returns the conversation ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// ItemId returns the item ID
func (m Model) ItemId() uint32 {
	return m.itemId
}

// NpcId returns the NPC ID the dialogue is rendered with
func (m Model) NpcId() uint32 {
	return m.npcId
}

// ScriptName returns the WZ spec/script value for authoring traceability
func (m Model) ScriptName() string {
	return m.scriptName
}

// StartState returns the starting state ID
func (m Model) StartState() string {
	return m.startState
}

// States returns the conversation states
func (m Model) States() []conversation.StateModel {
	return m.states
}

// CreatedAt returns the creation timestamp
func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m Model) UpdatedAt() time.Time {
	return m.updatedAt
}

// FindState implements conversation.StateContainer.
func (m Model) FindState(stateId string) (conversation.StateModel, error) {
	for _, state := range m.states {
		if state.Id() == stateId {
			return state, nil
		}
	}
	return conversation.StateModel{}, errors.New("state not found")
}
