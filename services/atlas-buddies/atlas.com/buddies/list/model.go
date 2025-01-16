package list

import (
	"atlas-buddies/buddy"
	"github.com/google/uuid"
)

type Model struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	capacity    uint32
	buddies     []buddy.Model
}

func (m Model) Buddies() []buddy.Model {
	return m.buddies
}

func (m Model) Capacity() uint32 {
	return m.capacity
}
