package equipslot

import (
	"time"

	"github.com/google/uuid"
)

// Model is a character's purchased equipped-inventory slot extension.
type Model struct {
	id          uuid.UUID
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

func modelFromEntity(e Entity) Model {
	return Model{
		id:          e.Id,
		characterId: e.CharacterId,
		slotIndex:   e.SlotIndex,
		expiresAt:   e.ExpiresAt,
	}
}
