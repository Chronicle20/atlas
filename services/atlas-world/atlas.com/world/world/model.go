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
	expRate            float64
	mesoRate           float64
	itemDropRate       float64
	questExpRate       float64
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

func (m Model) ExpRate() float64 {
	if m.expRate == 0 {
		return 1.0
	}
	return m.expRate
}

func (m Model) MesoRate() float64 {
	if m.mesoRate == 0 {
		return 1.0
	}
	return m.mesoRate
}

func (m Model) ItemDropRate() float64 {
	if m.itemDropRate == 0 {
		return 1.0
	}
	return m.itemDropRate
}

func (m Model) QuestExpRate() float64 {
	if m.questExpRate == 0 {
		return 1.0
	}
	return m.questExpRate
}
