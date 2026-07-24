package game

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Status represents the lifecycle state of an RPS session.
type Status string

const (
	StatusOpen             Status = "OPEN"
	StatusAwaitingSelect   Status = "AWAITING_SELECT"
	StatusAwaitingDecision Status = "AWAITING_DECISION"
	StatusEnded            Status = "ENDED"
)

// Throw represents a rock/paper/scissors selection.
type Throw byte

const (
	ThrowRock Throw = iota
	ThrowPaper
	ThrowScissors
)

// Model is an immutable representation of an RPS session, keyed by tenant + character.
type Model struct {
	tenant      tenant.Model
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	npcId       uint32
	rung        int
	status      Status
	lastThrow   Throw
	createdAt   time.Time
	updatedAt   time.Time
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Tenant      tenant.Model `json:"tenant"`
		CharacterId uint32       `json:"characterId"`
		WorldId     world.Id     `json:"worldId"`
		ChannelId   channel.Id   `json:"channelId"`
		NpcId       uint32       `json:"npcId"`
		Rung        int          `json:"rung"`
		Status      Status       `json:"status"`
		LastThrow   Throw        `json:"lastThrow"`
		CreatedAt   time.Time    `json:"createdAt"`
		UpdatedAt   time.Time    `json:"updatedAt"`
	}{
		Tenant:      m.tenant,
		CharacterId: m.characterId,
		WorldId:     m.worldId,
		ChannelId:   m.channelId,
		NpcId:       m.npcId,
		Rung:        m.rung,
		Status:      m.status,
		LastThrow:   m.lastThrow,
		CreatedAt:   m.createdAt,
		UpdatedAt:   m.updatedAt,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	t := &struct {
		Tenant      tenant.Model `json:"tenant"`
		CharacterId uint32       `json:"characterId"`
		WorldId     world.Id     `json:"worldId"`
		ChannelId   channel.Id   `json:"channelId"`
		NpcId       uint32       `json:"npcId"`
		Rung        int          `json:"rung"`
		Status      Status       `json:"status"`
		LastThrow   Throw        `json:"lastThrow"`
		CreatedAt   time.Time    `json:"createdAt"`
		UpdatedAt   time.Time    `json:"updatedAt"`
	}{}
	if err := json.Unmarshal(data, t); err != nil {
		return err
	}
	m.tenant = t.Tenant
	m.characterId = t.CharacterId
	m.worldId = t.WorldId
	m.channelId = t.ChannelId
	m.npcId = t.NpcId
	m.rung = t.Rung
	m.status = t.Status
	m.lastThrow = t.LastThrow
	m.createdAt = t.CreatedAt
	m.updatedAt = t.UpdatedAt
	return nil
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) NpcId() uint32 {
	return m.npcId
}

func (m Model) Rung() int {
	return m.rung
}

func (m Model) Status() Status {
	return m.status
}

func (m Model) LastThrow() Throw {
	return m.lastThrow
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) UpdatedAt() time.Time {
	return m.updatedAt
}
