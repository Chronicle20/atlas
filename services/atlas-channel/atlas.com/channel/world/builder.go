package world

import (
	"atlas-channel/channel"

	"github.com/Chronicle20/atlas-constants/world"
)

type modelBuilder struct {
	id                 world.Id
	name               string
	state              State
	message            string
	eventMessage       string
	recommendedMessage string
	capacityStatus     Status
	channels           []channel.Model
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder() *modelBuilder {
	return NewModelBuilder()
}

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
	}
}

func (b *modelBuilder) SetId(id world.Id) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetName(name string) *modelBuilder {
	b.name = name
	return b
}

func (b *modelBuilder) SetState(state State) *modelBuilder {
	b.state = state
	return b
}

func (b *modelBuilder) SetMessage(message string) *modelBuilder {
	b.message = message
	return b
}

func (b *modelBuilder) SetEventMessage(eventMessage string) *modelBuilder {
	b.eventMessage = eventMessage
	return b
}

func (b *modelBuilder) SetRecommendedMessage(recommendedMessage string) *modelBuilder {
	b.recommendedMessage = recommendedMessage
	return b
}

func (b *modelBuilder) SetCapacityStatus(capacityStatus Status) *modelBuilder {
	b.capacityStatus = capacityStatus
	return b
}

func (b *modelBuilder) SetChannels(channels []channel.Model) *modelBuilder {
	b.channels = channels
	return b
}

func (b *modelBuilder) Build() (Model, error) {
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

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
