package world

import (
	"atlas-world/channel"
)

type State byte
type Status uint16

const (
	StateNormal State = 0
	StateEvent  State = 1
	StateNew    State = 2
	StateHot    State = 3

	StatusNormal          Status = 0
	StatusHighlyPopulated Status = 1
	StatusFull            Status = 2
)

type Model struct {
	id                 byte
	name               string
	state              State
	message            string
	eventMessage       string
	recommendedMessage string
	capacityStatus     Status
	channels           []channel.Model
}

func (m Model) Id() byte {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) State() State {
	return m.state
}

func (m Model) Message() string {
	return m.message
}

func (m Model) EventMessage() string {
	return m.eventMessage
}

func (m Model) Recommended() bool {
	return m.recommendedMessage != ""
}

func (m Model) RecommendedMessage() string {
	return m.recommendedMessage
}

func (m Model) CapacityStatus() Status {
	return m.capacityStatus
}

func (m Model) Channels() []channel.Model {
	return m.channels
}

// Builder is used to construct a Model instance
type Builder struct {
	id                 byte
	name               string
	state              State
	message            string
	eventMessage       string
	recommendedMessage string
	capacityStatus     Status
	channels           []channel.Model
}

// NewBuilder creates a new Builder instance
func NewBuilder() *Builder {
	return &Builder{}
}

// SetId sets the id field
func (b *Builder) SetId(id byte) *Builder {
	b.id = id
	return b
}

// SetName sets the name field
func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

// SetState sets the state field
func (b *Builder) SetState(state State) *Builder {
	b.state = state
	return b
}

// SetMessage sets the message field
func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

// SetEventMessage sets the eventMessage field
func (b *Builder) SetEventMessage(eventMessage string) *Builder {
	b.eventMessage = eventMessage
	return b
}

// SetRecommendedMessage sets the recommendedMessage field
func (b *Builder) SetRecommendedMessage(recommendedMessage string) *Builder {
	b.recommendedMessage = recommendedMessage
	return b
}

// SetCapacityStatus sets the capacityStatus field
func (b *Builder) SetCapacityStatus(capacityStatus Status) *Builder {
	b.capacityStatus = capacityStatus
	return b
}

// SetChannels sets the channels field
func (b *Builder) SetChannels(channels []channel.Model) *Builder {
	b.channels = channels
	return b
}

// Build creates a new Model instance with the Builder's values
func (b *Builder) Build() Model {
	return Model{
		id:                 b.id,
		name:               b.name,
		state:              b.state,
		message:            b.message,
		eventMessage:       b.eventMessage,
		recommendedMessage: b.recommendedMessage,
		capacityStatus:     b.capacityStatus,
		channels:           b.channels,
	}
}

// ToBuilder creates a Builder initialized with the Model's values
func (m Model) ToBuilder() *Builder {
	return NewBuilder().
		SetId(m.id).
		SetName(m.name).
		SetState(m.state).
		SetMessage(m.message).
		SetEventMessage(m.eventMessage).
		SetRecommendedMessage(m.recommendedMessage).
		SetCapacityStatus(m.capacityStatus).
		SetChannels(m.channels)
}
