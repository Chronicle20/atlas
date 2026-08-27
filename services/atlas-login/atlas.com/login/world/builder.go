package world

import (
	"atlas-login/channel"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder is used to construct a Model instance
type Builder struct {
	id                 world.Id
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
func (b *Builder) SetId(id world.Id) *Builder {
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
