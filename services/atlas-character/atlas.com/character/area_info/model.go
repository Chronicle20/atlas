package area_info

import (
	"github.com/google/uuid"
)

type Model struct {
	id          uuid.UUID
	characterId uint32
	area        uint16
	info        string
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Area() uint16 {
	return m.area
}

func (m Model) Info() string {
	return m.info
}
