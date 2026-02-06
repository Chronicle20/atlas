package list

import (
	"atlas-buddies/buddy"

	"github.com/google/uuid"
)

type Model struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	capacity    byte
	buddies     []buddy.Model
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Capacity() byte {
	return m.capacity
}

func (m Model) Buddies() []buddy.Model {
	return m.buddies
}
