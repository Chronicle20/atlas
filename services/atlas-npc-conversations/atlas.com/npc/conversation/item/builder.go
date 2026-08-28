package item

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder is a builder for Model
type Builder struct {
	id         uuid.UUID
	itemId     uint32
	npcId      uint32
	scriptName string
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

// SetItemId sets the item ID
func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

// SetNpcId sets the NPC ID
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetScriptName sets the WZ spec/script value
func (b *Builder) SetScriptName(scriptName string) *Builder {
	b.scriptName = scriptName
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
func (b *Builder) SetCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

// SetUpdatedAt sets the last update timestamp
func (b *Builder) SetUpdatedAt(t time.Time) *Builder {
	b.updatedAt = t
	return b
}

// Build builds the Model
func (b *Builder) Build() (Model, error) {
	if b.itemId == 0 {
		return Model{}, errors.New("itemId is required")
	}
	if b.startState == "" {
		return Model{}, errors.New("startState is required")
	}
	if len(b.states) == 0 {
		return Model{}, errors.New("at least one state is required")
	}

	return Model{
		id:         b.id,
		itemId:     b.itemId,
		npcId:      b.npcId,
		scriptName: b.scriptName,
		startState: b.startState,
		states:     b.states,
		createdAt:  b.createdAt,
		updatedAt:  b.updatedAt,
	}, nil
}
