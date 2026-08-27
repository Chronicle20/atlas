package quest

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Model represents a quest conversation with dual state machines
type Model struct {
	id                uuid.UUID
	questId           uint32
	npcId             uint32 // Metadata: NPC that gives this quest
	questName         string // Metadata: Human-readable quest name
	startStateMachine StateMachine
	endStateMachine   *StateMachine // Optional: nil if quest only has start dialogue
	createdAt         time.Time
	updatedAt         time.Time
}

// Id returns the conversation ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// QuestId returns the quest ID
func (m Model) QuestId() uint32 {
	return m.questId
}

// NpcId returns the NPC ID (metadata)
func (m Model) NpcId() uint32 {
	return m.npcId
}

// QuestName returns the quest name (metadata)
func (m Model) QuestName() string {
	return m.questName
}

// StartStateMachine returns the state machine for quest acceptance
func (m Model) StartStateMachine() StateMachine {
	return m.startStateMachine
}

// EndStateMachine returns the state machine for quest completion (may be nil)
func (m Model) EndStateMachine() *StateMachine {
	return m.endStateMachine
}

// HasEndStateMachine returns true if this quest has an end state machine
func (m Model) HasEndStateMachine() bool {
	return m.endStateMachine != nil
}

// CreatedAt returns the creation timestamp
func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m Model) UpdatedAt() time.Time {
	return m.updatedAt
}

// FindStateInStartMachine finds a state by ID in the start state machine
func (m Model) FindStateInStartMachine(stateId string) (conversation.StateModel, error) {
	return m.startStateMachine.FindState(stateId)
}

// FindStateInEndMachine finds a state by ID in the end state machine
func (m Model) FindStateInEndMachine(stateId string) (conversation.StateModel, error) {
	if m.endStateMachine == nil {
		return conversation.StateModel{}, errors.New("end state machine is nil")
	}
	return m.endStateMachine.FindState(stateId)
}

// StateMachine represents a state machine within a quest conversation
type StateMachine struct {
	startState string
	states     []conversation.StateModel
}

// StartState returns the starting state ID
func (s StateMachine) StartState() string {
	return s.startState
}

// States returns the conversation states
func (s StateMachine) States() []conversation.StateModel {
	return s.states
}

// FindState finds a state by ID
func (s StateMachine) FindState(stateId string) (conversation.StateModel, error) {
	for _, state := range s.states {
		if state.Id() == stateId {
			return state, nil
		}
	}
	return conversation.StateModel{}, errors.New("state not found")
}
