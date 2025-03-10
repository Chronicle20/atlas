package pet

import "time"

type Model struct {
	id              uint32
	inventoryItemId uint32
	templateId      uint32
	name            string
	level           byte
	tameness        uint16
	fullness        byte
	expiration      time.Time
	ownerId         uint32
	lead            bool
	slot            byte
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) InventoryItemId() uint32 {
	return m.inventoryItemId
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Tameness() uint16 {
	return m.tameness
}

func (m Model) Fullness() byte {
	return m.fullness
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) Lead() bool {
	return m.lead
}

func (m Model) Slot() byte {
	return m.slot
}
