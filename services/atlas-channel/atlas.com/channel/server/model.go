package server

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type Model struct {
	tenant    tenant.Model
	ch        channel.Model
	ipAddress string
	port      int
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) Channel() channel.Model {
	return m.ch
}

func (m Model) WorldId() world.Id {
	return m.Channel().WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.Channel().Id()
}

func (m Model) IpAddress() string {
	return m.ipAddress
}

func (m Model) Port() int {
	return m.port
}

func (m Model) String() string {
	return fmt.Sprintf("Tenant [%s] World [%d] Channel [%d]", m.tenant.String(), m.Channel().WorldId(), m.Channel().Id())
}

func (m Model) Is(t tenant.Model, worldId world.Id, channelId channel.Id) bool {
	if !m.IsWorld(t, worldId) {
		return false
	}
	if channelId != m.Channel().Id() {
		return false
	}
	return true
}

func (m Model) IsWorld(t tenant.Model, worldId world.Id) bool {
	is := t.Is(m.Tenant())
	if !is {
		return false
	}
	if worldId != m.Channel().WorldId() {
		return false
	}
	return true
}

func (m Model) Map(mapId _map.Id) _map.Model {
	return _map.NewModel(m.WorldId())(m.ChannelId())(mapId)
}

func (m Model) Field(mapId _map.Id, instance uuid.UUID) field.Model {
	return field.NewBuilder(m.WorldId(), m.ChannelId(), mapId).SetInstance(instance).Build()
}
