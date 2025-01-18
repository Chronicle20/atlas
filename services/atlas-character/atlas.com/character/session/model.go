package session

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"time"
)

type State uint8

const (
	StateLoggedOut  = State(0)
	StateLoggedIn   = State(1)
	StateTransition = State(2)
)

type Model struct {
	tenant      tenant.Model
	characterId uint32
	worldId     byte
	channelId   byte
	state       State
	age         time.Time
}

func (m Model) State() State {
	return m.state
}

func (m Model) Age() time.Time {
	return m.age
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}
