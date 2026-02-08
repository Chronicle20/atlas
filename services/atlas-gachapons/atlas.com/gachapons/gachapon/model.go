package gachapon

import "github.com/google/uuid"

type Model struct {
	tenantId       uuid.UUID
	id             string
	name           string
	npcIds         []uint32
	commonWeight   uint32
	uncommonWeight uint32
	rareWeight     uint32
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() string {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) NpcIds() []uint32 {
	return m.npcIds
}

func (m Model) CommonWeight() uint32 {
	return m.commonWeight
}

func (m Model) UncommonWeight() uint32 {
	return m.uncommonWeight
}

func (m Model) RareWeight() uint32 {
	return m.rareWeight
}
