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
