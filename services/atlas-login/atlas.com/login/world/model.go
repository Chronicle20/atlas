package world

import (
	"atlas-login/channel"
	"github.com/Chronicle20/atlas-model/model"
)

type Model struct {
	id                 byte
	name               string
	state              State
	message            string
	eventMessage       string
	recommended        bool
	recommendedMessage string
	capacityStatus     Status
	channelLoad        []channel.Load
}

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

func (w Model) Id() byte {
	return w.id
}

func (w Model) SetChannelLoad(val []channel.Load) Model {
	return CloneWorld(w).
		SetChannelLoad(val).
		Build()
}

func (w Model) Name() string {
	return w.name
}

func (w Model) State() State {
	return w.state
}

func (w Model) EventMessage() string {
	return w.eventMessage
}

func (w Model) ChannelLoad() []channel.Load {
	return w.channelLoad
}

func (w Model) Recommended() bool {
	return w.recommended
}

func (w Model) Recommendation() Recommendation {
	return NewWorldRecommendation(w.id, w.recommendedMessage)
}

func (w Model) CapacityStatus() Status {
	return w.capacityStatus
}

func Clone(m Model) model.Provider[Model] {
	return func() (Model, error) {
		return CloneWorld(m).Build(), nil
	}
}

type worldBuilder struct {
	id                 byte
	name               string
	state              State
	message            string
	eventMessage       string
	recommended        bool
	recommendedMessage string
	capacityStatus     Status
	channelLoad        []channel.Load
}

func NewWorldBuilder() *worldBuilder {
	return &worldBuilder{}
}

func CloneWorld(o Model) *worldBuilder {
	return &worldBuilder{
		id:                 o.id,
		name:               o.name,
		state:              o.state,
		message:            o.message,
		eventMessage:       o.eventMessage,
		recommended:        o.recommended,
		recommendedMessage: o.recommendedMessage,
		capacityStatus:     o.capacityStatus,
		channelLoad:        o.channelLoad,
	}
}

func (w *worldBuilder) SetId(id byte) *worldBuilder {
	w.id = id
	return w
}

func (w *worldBuilder) SetName(name string) *worldBuilder {
	w.name = name
	return w
}

func (w *worldBuilder) SetState(state State) *worldBuilder {
	w.state = state
	return w
}

func (w *worldBuilder) SetMessage(message string) *worldBuilder {
	w.message = message
	return w
}

func (w *worldBuilder) SetEventMessage(eventMessage string) *worldBuilder {
	w.eventMessage = eventMessage
	return w
}

func (w *worldBuilder) SetRecommended(recommended bool) *worldBuilder {
	w.recommended = recommended
	return w
}

func (w *worldBuilder) SetRecommendedMessage(recommendedMessage string) *worldBuilder {
	w.recommendedMessage = recommendedMessage
	return w
}

func (w *worldBuilder) SetCapacityStatus(capacityStatus Status) *worldBuilder {
	w.capacityStatus = capacityStatus
	return w
}

func (w *worldBuilder) AddChannelLoad(channelId byte, capacity int) *worldBuilder {
	w.channelLoad = append(w.channelLoad, channel.NewChannelLoad(channelId, capacity))
	return w
}

func (w *worldBuilder) SetChannelLoad(channelLoad []channel.Load) *worldBuilder {
	w.channelLoad = channelLoad
	return w
}

func (w *worldBuilder) Build() Model {
	return Model{
		id:                 w.id,
		name:               w.name,
		state:              w.state,
		message:            w.message,
		eventMessage:       w.eventMessage,
		recommended:        w.recommended,
		recommendedMessage: w.recommendedMessage,
		capacityStatus:     w.capacityStatus,
		channelLoad:        w.channelLoad,
	}
}

type Recommendation struct {
	worldId byte
	reason  string
}

func (r Recommendation) WorldId() byte {
	return r.worldId
}

func (r Recommendation) Reason() string {
	return r.reason
}

func NewWorldRecommendation(worldId byte, reason string) Recommendation {
	return Recommendation{worldId, reason}
}
