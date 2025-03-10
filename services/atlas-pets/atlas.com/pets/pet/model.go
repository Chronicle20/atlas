package pet

import "time"

type Model struct {
	id              uint64
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

func (m Model) Id() uint64 {
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

type ModelBuilder struct {
	id              uint64
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

func NewModelBuilder(id uint64, inventoryItemId uint32, templateId uint32, name string, ownerId uint32) *ModelBuilder {
	return &ModelBuilder{
		id:              id,
		inventoryItemId: inventoryItemId,
		templateId:      templateId,
		name:            name,
		level:           1,
		tameness:        0,
		fullness:        100,
		expiration:      time.Now().Add(720 * time.Hour),
		ownerId:         ownerId,
		lead:            false,
		slot:            0,
	}
}

func (b *ModelBuilder) SetLevel(level byte) *ModelBuilder {
	b.level = level
	return b
}

func (b *ModelBuilder) SetTameness(tameness uint16) *ModelBuilder {
	b.tameness = tameness
	return b
}

func (b *ModelBuilder) SetFullness(fullness byte) *ModelBuilder {
	b.fullness = fullness
	return b
}

func (b *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	b.expiration = expiration
	return b
}

func (b *ModelBuilder) SetSlot(slot byte) *ModelBuilder {
	b.slot = slot
	return b
}

func (b *ModelBuilder) SetLead(lead bool) *ModelBuilder {
	b.lead = lead
	return b
}

// Build returns the constructed Model instance
func (b *ModelBuilder) Build() Model {
	return Model{
		id:              b.id,
		inventoryItemId: b.inventoryItemId,
		templateId:      b.templateId,
		name:            b.name,
		level:           b.level,
		tameness:        b.tameness,
		fullness:        b.fullness,
		expiration:      b.expiration,
		ownerId:         b.ownerId,
		lead:            b.lead,
		slot:            b.slot,
	}
}
