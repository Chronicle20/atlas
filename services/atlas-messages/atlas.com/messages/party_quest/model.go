package party_quest

import "github.com/google/uuid"

type Model struct {
	id uuid.UUID
}

func (m Model) Id() uuid.UUID {
	return m.id
}
