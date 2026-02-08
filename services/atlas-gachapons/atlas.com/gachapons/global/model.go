package global

import "github.com/google/uuid"

type Model struct {
	tenantId uuid.UUID
	id       uint32
	itemId   uint32
	quantity uint32
	tier     string
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Tier() string {
	return m.tier
}
