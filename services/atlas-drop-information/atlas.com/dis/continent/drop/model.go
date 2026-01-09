package drop

import "github.com/google/uuid"

type Model struct {
	tenantId        uuid.UUID
	id              uint32
	continentId     int32
	itemId          uint32
	minimumQuantity uint32
	maximumQuantity uint32
	questId         uint32
	chance          uint32
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ContinentId() int32 {
	return m.continentId
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) MinimumQuantity() uint32 {
	return m.minimumQuantity
}

func (m Model) MaximumQuantity() uint32 {
	return m.maximumQuantity
}

func (m Model) QuestId() uint32 {
	return m.questId
}

func (m Model) Chance() uint32 {
	return m.chance
}
