package npc

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"time"

	"github.com/google/uuid"
)

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
