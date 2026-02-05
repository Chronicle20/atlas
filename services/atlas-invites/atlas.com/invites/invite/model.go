package invite

import (
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type Model struct {
	tenant       tenant.Model
	id           uint32
	inviteType   string
	referenceId  uint32
	originatorId uint32
	targetId     uint32
	worldId      world.Id
	age          time.Time
}

func (m Model) ReferenceId() uint32 {
	return m.referenceId
}

func (m Model) TargetId() uint32 {
	return m.targetId
}

func (m Model) OriginatorId() uint32 {
	return m.originatorId
}

func (m Model) Expired(timeout time.Duration) bool {
	if time.Now().Sub(m.Age()) > timeout {
		return true
	}
	return false
}

func (m Model) Age() time.Time {
	return m.age
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) Type() string {
	return m.inviteType
}

func (m Model) WorldId() world.Id {
	return m.worldId
}
