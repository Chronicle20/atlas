package server

import (
	"fmt"
	tenant "github.com/Chronicle20/atlas-tenant"
)

type Model struct {
	tenant    tenant.Model
	worldId   byte
	channelId byte
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

func (m Model) String() string {
	return fmt.Sprintf("Tenant [%s] World [%d] Channel [%d]", m.tenant.String(), m.worldId, m.channelId)
}

func New(tenant tenant.Model, worldId byte, channelId byte) (Model, error) {
	return Model{
		tenant:    tenant,
		worldId:   worldId,
		channelId: channelId,
	}, nil
}

func (m Model) Is(tenant tenant.Model, worldId byte, channelId byte) bool {
	t := m.Tenant()
	is := t.Is(tenant)
	if !is {
		return false
	}
	if worldId != m.worldId {
		return false
	}
	if channelId != m.channelId {
		return false
	}
	return true
}
