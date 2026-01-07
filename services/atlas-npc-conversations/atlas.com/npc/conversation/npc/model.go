package npc

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"github.com/google/uuid"
	"time"
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

// Builder is a builder for Model
type Builder struct {
	id         uuid.UUID
	npcId      uint32
	startState string
	states     []conversation.StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		id:        uuid.Nil,
		states:    make([]conversation.StateModel, 0),
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// SetId sets the conversation ID
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetNpcId sets the NPC ID
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetStartState sets the starting state ID
func (b *Builder) SetStartState(startState string) *Builder {
	b.startState = startState
	return b
}

// SetStates sets the conversation states
func (b *Builder) SetStates(states []conversation.StateModel) *Builder {
	b.states = states
	return b
}

// AddState adds a conversation state
func (b *Builder) AddState(state conversation.StateModel) *Builder {
	b.states = append(b.states, state)
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
	if b.npcId == 0 {
		return Model{}, errors.New("npcId is required")
	}
	if b.startState == "" {
		return Model{}, errors.New("startState is required")
	}
	if len(b.states) == 0 {
		return Model{}, errors.New("at least one state is required")
	}

	return Model{
		id:         b.id,
		npcId:      b.npcId,
		startState: b.startState,
		states:     b.states,
		createdAt:  b.createdAt,
		updatedAt:  b.updatedAt,
	}, nil
}
