package wallet

import "github.com/google/uuid"

type Model struct {
	id        uuid.UUID
	accountId uint32
	credit    uint32
	points    uint32
	prepaid   uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) AccountId() uint32 {
	return m.accountId
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
