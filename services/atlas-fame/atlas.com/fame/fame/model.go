package fame

import (
	"github.com/google/uuid"
	"time"
)

type Model struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	targetId    uint32
	amount      int8
	createdAt   time.Time
}

func (m Model) TargetId() uint32 {
	return m.targetId
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}
