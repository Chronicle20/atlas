package quest

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"github.com/google/uuid"
	"time"
)

// Model represents a quest conversation with dual state machines
type Model struct {
	id                uuid.UUID
	questId           uint32
	npcId             uint32      // Metadata: NPC that gives this quest
	questName         string      // Metadata: Human-readable quest name
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

// Builder is a builder for Model
type Builder struct {
	id                uuid.UUID
	questId           uint32
	npcId             uint32
	questName         string
	startStateMachine StateMachine
	endStateMachine   *StateMachine
	createdAt         time.Time
	updatedAt         time.Time
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		id:        uuid.Nil,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// SetId sets the conversation ID
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetQuestId sets the quest ID
func (b *Builder) SetQuestId(questId uint32) *Builder {
	b.questId = questId
	return b
}

// SetNpcId sets the NPC ID (metadata)
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetQuestName sets the quest name (metadata)
func (b *Builder) SetQuestName(questName string) *Builder {
	b.questName = questName
	return b
}

// SetStartStateMachine sets the start state machine
func (b *Builder) SetStartStateMachine(sm StateMachine) *Builder {
	b.startStateMachine = sm
	return b
}

// SetEndStateMachine sets the end state machine
func (b *Builder) SetEndStateMachine(sm *StateMachine) *Builder {
	b.endStateMachine = sm
	return b
}

// SetCreatedAt sets the creation timestamp
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the last update timestamp
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

// Build builds the Model
func (b *Builder) Build() (Model, error) {
	if b.questId == 0 {
		return Model{}, errors.New("questId is required")
	}
	if b.startStateMachine.startState == "" {
		return Model{}, errors.New("startStateMachine.startState is required")
	}
	if len(b.startStateMachine.states) == 0 {
		return Model{}, errors.New("startStateMachine must have at least one state")
	}

	return Model{
		id:                b.id,
		questId:           b.questId,
		npcId:             b.npcId,
		questName:         b.questName,
		startStateMachine: b.startStateMachine,
		endStateMachine:   b.endStateMachine,
		createdAt:         b.createdAt,
		updatedAt:         b.updatedAt,
	}, nil
}

// StateMachineBuilder is a builder for StateMachine
type StateMachineBuilder struct {
	startState string
	states     []conversation.StateModel
}

// NewStateMachineBuilder creates a new StateMachineBuilder
func NewStateMachineBuilder() *StateMachineBuilder {
	return &StateMachineBuilder{
		states: make([]conversation.StateModel, 0),
	}
}

// SetStartState sets the starting state ID
func (b *StateMachineBuilder) SetStartState(startState string) *StateMachineBuilder {
	b.startState = startState
	return b
}

// SetStates sets the conversation states
func (b *StateMachineBuilder) SetStates(states []conversation.StateModel) *StateMachineBuilder {
	b.states = states
	return b
}

// AddState adds a conversation state
func (b *StateMachineBuilder) AddState(state conversation.StateModel) *StateMachineBuilder {
	b.states = append(b.states, state)
	return b
}

// Build builds the StateMachine
func (b *StateMachineBuilder) Build() (StateMachine, error) {
	if b.startState == "" {
		return StateMachine{}, errors.New("startState is required")
	}
	if len(b.states) == 0 {
		return StateMachine{}, errors.New("at least one state is required")
	}

	return StateMachine{
		startState: b.startState,
		states:     b.states,
	}, nil
}
