package buddylist

import (
	"atlas-channel/buddylist/buddy"
	"github.com/google/uuid"
)

type Model struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	capacity    byte
	buddies     []buddy.Model
}

func (m Model) Buddies() []buddy.Model {
	return m.buddies
}

func (m Model) Capacity() byte {
	return m.capacity
}
