package server

import (
	"atlas-channel/tenant"
	"fmt"
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
	if tenant.Id != m.tenant.Id {
		return false
	}
	if tenant.Region != m.tenant.Region {
		return false
	}
	if tenant.MajorVersion != m.tenant.MajorVersion {
		return false
	}
	if tenant.MinorVersion != m.tenant.MinorVersion {
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
