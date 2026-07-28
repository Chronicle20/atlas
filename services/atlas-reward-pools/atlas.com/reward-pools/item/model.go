package item

import "github.com/google/uuid"

type Model struct {
	tenantId   uuid.UUID
	id         uint32
	gachaponId string
	itemId     uint32
	quantity   uint32
	tier       string
	weight     uint32
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) GachaponId() string {
	return m.gachaponId
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

// Weight is an optional explicit roll weight for this item, used by
// weighted (e.g. incubator) reward pools. Items that never set a weight
// read 0; the existing tier-based roll does not consume this value.
func (m Model) Weight() uint32 {
	return m.weight
}
