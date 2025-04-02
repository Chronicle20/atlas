package wishlist

import "github.com/google/uuid"

type Model struct {
	id           uuid.UUID
	characterId  uint32
	serialNumber uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) SerialNumber() uint32 {
	return m.serialNumber
}
