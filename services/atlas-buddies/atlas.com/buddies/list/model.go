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
