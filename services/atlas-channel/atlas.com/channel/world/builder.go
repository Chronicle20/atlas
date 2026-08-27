package world

import (
	"atlas-channel/channel"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type builder struct {
	id                 world.Id
	name               string
	state              State
	message            string
	eventMessage       string
	recommendedMessage string
	capacityStatus     Status
	channels           []channel.Model
}

// NewBuilder creates a new builder instance
func NewBuilder() *builder {
	return &builder{}
}

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
	}
}

func (b *builder) SetId(id world.Id) *builder {
	b.id = id
	return b
}

func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

func (b *builder) SetState(state State) *builder {
	b.state = state
	return b
}

func (b *builder) SetMessage(message string) *builder {
	b.message = message
	return b
}

func (b *builder) SetEventMessage(eventMessage string) *builder {
	b.eventMessage = eventMessage
	return b
}

func (b *builder) SetRecommendedMessage(recommendedMessage string) *builder {
	b.recommendedMessage = recommendedMessage
	return b
}

func (b *builder) SetCapacityStatus(capacityStatus Status) *builder {
	b.capacityStatus = capacityStatus
	return b
}

func (b *builder) SetChannels(channels []channel.Model) *builder {
	b.channels = channels
	return b
}

func (b *builder) Build() (Model, error) {
	return Model{
		id:                 b.id,
		name:               b.name,
		state:              b.state,
		message:            b.message,
		eventMessage:       b.eventMessage,
		recommendedMessage: b.recommendedMessage,
		capacityStatus:     b.capacityStatus,
		channels:           b.channels,
	}, nil
}

func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
