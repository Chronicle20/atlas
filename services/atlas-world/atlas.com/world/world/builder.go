package world

import (
	"atlas-world/channel"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

var ErrMissingName = errors.New("world name is required")

type builder struct {
	id                 world.Id
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

// NewBuilder creates a new builder instance
func NewBuilder() *builder {
	return &builder{
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *builder {
	return &builder{
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
func (b *builder) SetId(id world.Id) *builder {
	b.id = id
	return b
}

// SetName sets the name field
func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

// SetState sets the state field
func (b *builder) SetState(state State) *builder {
	b.state = state
	return b
}

// SetMessage sets the message field
func (b *builder) SetMessage(message string) *builder {
	b.message = message
	return b
}

// SetEventMessage sets the eventMessage field
func (b *builder) SetEventMessage(eventMessage string) *builder {
	b.eventMessage = eventMessage
	return b
}

// SetRecommendedMessage sets the recommendedMessage field
func (b *builder) SetRecommendedMessage(recommendedMessage string) *builder {
	b.recommendedMessage = recommendedMessage
	return b
}

// SetCapacityStatus sets the capacityStatus field
func (b *builder) SetCapacityStatus(capacityStatus Status) *builder {
	b.capacityStatus = capacityStatus
	return b
}

// SetChannels sets the channels field
func (b *builder) SetChannels(channels []channel.Model) *builder {
	b.channels = channels
	return b
}

// SetExpRate sets the experience rate multiplier
func (b *builder) SetExpRate(expRate float64) *builder {
	b.expRate = expRate
	return b
}

// SetMesoRate sets the meso rate multiplier
func (b *builder) SetMesoRate(mesoRate float64) *builder {
	b.mesoRate = mesoRate
	return b
}

// SetItemDropRate sets the item drop rate multiplier
func (b *builder) SetItemDropRate(itemDropRate float64) *builder {
	b.itemDropRate = itemDropRate
	return b
}

// SetQuestExpRate sets the quest experience rate multiplier
func (b *builder) SetQuestExpRate(questExpRate float64) *builder {
	b.questExpRate = questExpRate
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
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
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
