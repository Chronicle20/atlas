package wallet

import "github.com/google/uuid"

type Model struct {
	id          uuid.UUID
	characterId uint32
	credit      uint32
	points      uint32
	prepaid     uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Credit() uint32 {
	return m.credit
}

func (m Model) Points() uint32 {
	return m.points
}

func (m Model) Prepaid() uint32 {
	return m.prepaid
}
