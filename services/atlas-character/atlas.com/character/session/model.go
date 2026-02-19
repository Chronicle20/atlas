package session

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
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
	worldId     world.Id
	channelId   channel.Id
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

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

type jsonModel struct {
	Tenant      *tenant.Model `json:"tenant"`
	CharacterId uint32        `json:"characterId"`
	WorldId     world.Id      `json:"worldId"`
	ChannelId   channel.Id    `json:"channelId"`
	State       State         `json:"state"`
	Age         time.Time     `json:"age"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	t := m.tenant
	return json.Marshal(jsonModel{
		Tenant:      &t,
		CharacterId: m.characterId,
		WorldId:     m.worldId,
		ChannelId:   m.channelId,
		State:       m.state,
		Age:         m.age,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j jsonModel
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	if j.Tenant != nil {
		m.tenant = *j.Tenant
	}
	m.characterId = j.CharacterId
	m.worldId = j.WorldId
	m.channelId = j.ChannelId
	m.state = j.State
	m.age = j.Age
	return nil
}
