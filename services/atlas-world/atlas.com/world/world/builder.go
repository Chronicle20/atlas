package world

import (
	"atlas-world/channel"
	"errors"
)

var (
	ErrMissingName = errors.New("world name is required")
)

type modelBuilder struct {
	id                 byte
	name               string
	state              State
	message            string
	eventMessage       string
	recommendedMessage string
	capacityStatus     Status
	channels           []channel.Model
	expRate            float64
	mesoRate           float64
	itemDropRate       float64
	questExpRate       float64
}

// NewModelBuilder creates a new builder instance
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:                 m.id,
		name:               m.name,
		state:              m.state,
		message:            m.message,
		eventMessage:       m.eventMessage,
		recommendedMessage: m.recommendedMessage,
		capacityStatus:     m.capacityStatus,
		channels:           m.channels,
		expRate:            m.expRate,
		mesoRate:           m.mesoRate,
		itemDropRate:       m.itemDropRate,
		questExpRate:       m.questExpRate,
	}
}

// SetId sets the id field
func (b *modelBuilder) SetId(id byte) *modelBuilder {
	b.id = id
	return b
}

// SetName sets the name field
func (b *modelBuilder) SetName(name string) *modelBuilder {
	b.name = name
	return b
}

// SetState sets the state field
func (b *modelBuilder) SetState(state State) *modelBuilder {
	b.state = state
	return b
}

// SetMessage sets the message field
func (b *modelBuilder) SetMessage(message string) *modelBuilder {
	b.message = message
	return b
}

// SetEventMessage sets the eventMessage field
func (b *modelBuilder) SetEventMessage(eventMessage string) *modelBuilder {
	b.eventMessage = eventMessage
	return b
}

// SetRecommendedMessage sets the recommendedMessage field
func (b *modelBuilder) SetRecommendedMessage(recommendedMessage string) *modelBuilder {
	b.recommendedMessage = recommendedMessage
	return b
}

// SetCapacityStatus sets the capacityStatus field
func (b *modelBuilder) SetCapacityStatus(capacityStatus Status) *modelBuilder {
	b.capacityStatus = capacityStatus
	return b
}

// SetChannels sets the channels field
func (b *modelBuilder) SetChannels(channels []channel.Model) *modelBuilder {
	b.channels = channels
	return b
}

// SetExpRate sets the experience rate multiplier
func (b *modelBuilder) SetExpRate(expRate float64) *modelBuilder {
	b.expRate = expRate
	return b
}

// SetMesoRate sets the meso rate multiplier
func (b *modelBuilder) SetMesoRate(mesoRate float64) *modelBuilder {
	b.mesoRate = mesoRate
	return b
}

// SetItemDropRate sets the item drop rate multiplier
func (b *modelBuilder) SetItemDropRate(itemDropRate float64) *modelBuilder {
	b.itemDropRate = itemDropRate
	return b
}

// SetQuestExpRate sets the quest experience rate multiplier
func (b *modelBuilder) SetQuestExpRate(questExpRate float64) *modelBuilder {
	b.questExpRate = questExpRate
	return b
}

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.name == "" {
		return Model{}, ErrMissingName
	}
	return Model{
		id:                 b.id,
		name:               b.name,
		state:              b.state,
		message:            b.message,
		eventMessage:       b.eventMessage,
		recommendedMessage: b.recommendedMessage,
		capacityStatus:     b.capacityStatus,
		channels:           b.channels,
		expRate:            b.expRate,
		mesoRate:           b.mesoRate,
		itemDropRate:       b.itemDropRate,
		questExpRate:       b.questExpRate,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
