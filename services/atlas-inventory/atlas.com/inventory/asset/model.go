package asset

import "time"

type Model struct {
	id            uint32
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) ReferenceId() uint32 {
	return m.referenceId
}

func (m Model) ReferenceType() string {
	return m.referenceType
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:            m.id,
		slot:          m.slot,
		templateId:    m.templateId,
		expiration:    m.expiration,
		referenceId:   m.referenceId,
		referenceType: m.referenceType,
	}
}

type ModelBuilder struct {
	id            uint32
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType string
}

func NewBuilder(id uint32, templateId uint32, referenceId uint32, referenceType string) *ModelBuilder {
	return &ModelBuilder{
		id:            id,
		slot:          0,
		templateId:    templateId,
		expiration:    time.Time{},
		referenceId:   referenceId,
		referenceType: referenceType,
	}
}

func (b *ModelBuilder) SetSlot(slot int16) *ModelBuilder {
	b.slot = slot
	return b
}

func (b *ModelBuilder) SetExpiration(e time.Time) *ModelBuilder {
	b.expiration = e
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:            b.id,
		slot:          b.slot,
		templateId:    b.templateId,
		expiration:    b.expiration,
		referenceId:   b.referenceId,
		referenceType: b.referenceType,
	}
}
