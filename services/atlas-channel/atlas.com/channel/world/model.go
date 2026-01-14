package world

import (
	"atlas-channel/channel"
	"github.com/Chronicle20/atlas-constants/world"
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
	id                 world.Id
	name               string
	state              State
	message            string
	eventMessage       string
	recommendedMessage string
	capacityStatus     Status
	channels           []channel.Model
}

func (m Model) Id() world.Id {
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
