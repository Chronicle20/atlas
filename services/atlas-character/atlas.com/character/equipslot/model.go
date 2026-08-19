package equipslot

import (
	"time"

	"github.com/google/uuid"
)

// Model is a character's purchased equipped-inventory slot extension.
//
// tenantId is carried privately so ToEntity (entity.go) can round-trip a
// Model back into its persisted Entity without a caller re-supplying it; it
// is never part of RestModel (rest.go), matching every other package's REST
// surface -- a tenant identifier travels in context/headers only.
type Model struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	characterId uint32
	slotIndex   int16
	expiresAt   time.Time
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) SlotIndex() int16 {
	return m.slotIndex
}

func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}
