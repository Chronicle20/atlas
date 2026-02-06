package fame

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	targetId    uint32
	amount      int8
	createdAt   time.Time
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) TargetId() uint32 {
	return m.targetId
}

func (m Model) Amount() int8 {
	return m.amount
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}
