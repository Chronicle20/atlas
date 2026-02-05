package coordinator

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type Model struct {
	tenant    tenant.Model
	channel   channel.Model
	leaderId  uint32
	name      string
	requests  []uint32
	responses map[uint32]bool
	age       time.Time
}

func (m Model) Agree(characterId uint32) Model {
	m.responses[characterId] = true
	return Model{
		tenant:    m.tenant,
		channel:   m.channel,
		leaderId:  m.leaderId,
		name:      m.name,
		requests:  m.requests,
		responses: m.responses,
		age:       m.age,
	}
}

func (m Model) Responses() map[uint32]bool {
	return m.responses
}

func (m Model) Requests() []uint32 {
	return m.requests
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) WorldId() world.Id {
	return m.Channel().WorldId()
}

func (m Model) Channel() channel.Model {
	return m.channel
}

func (m Model) ChannelId() channel.Id {
	return m.Channel().Id()
}

func (m Model) Age() time.Time {
	return m.age
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}
