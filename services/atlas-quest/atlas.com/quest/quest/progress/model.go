package progress

import "github.com/google/uuid"

type Model struct {
	tenantId   uuid.UUID
	id         uint32
	infoNumber uint32
	progress   string
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) InfoNumber() uint32 {
	return m.infoNumber
}

func (m Model) Progress() string {
	return m.progress
}
