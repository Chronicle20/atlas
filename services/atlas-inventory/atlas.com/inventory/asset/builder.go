package asset

import (
	"time"

	"github.com/google/uuid"
)

func Clone[E any](m Model[E]) *ModelBuilder[E] {
	return &ModelBuilder[E]{
		id:            m.id,
		compartmentId: m.compartmentId,
		slot:          m.slot,
		templateId:    m.templateId,
		expiration:    m.expiration,
		referenceId:   m.referenceId,
		referenceType: m.referenceType,
		referenceData: m.referenceData,
	}
}

type ModelBuilder[E any] struct {
	id            uint32
	compartmentId uuid.UUID
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType ReferenceType
	referenceData E
}

func NewBuilder[E any](id uint32, compartmentId uuid.UUID, templateId uint32, referenceId uint32, referenceType ReferenceType) *ModelBuilder[E] {
	return &ModelBuilder[E]{
		id:            id,
		compartmentId: compartmentId,
		slot:          0,
		templateId:    templateId,
		expiration:    time.Time{},
		referenceId:   referenceId,
		referenceType: referenceType,
	}
}

func (b *ModelBuilder[E]) SetSlot(slot int16) *ModelBuilder[E] {
	b.slot = slot
	return b
}

func (b *ModelBuilder[E]) SetExpiration(e time.Time) *ModelBuilder[E] {
	b.expiration = e
	return b
}

func (b *ModelBuilder[E]) SetReferenceData(e E) *ModelBuilder[E] {
	b.referenceData = e
	return b
}

func (b *ModelBuilder[E]) Build() Model[E] {
	return Model[E]{
		id:            b.id,
		compartmentId: b.compartmentId,
		slot:          b.slot,
		templateId:    b.templateId,
		expiration:    b.expiration,
		referenceId:   b.referenceId,
		referenceType: b.referenceType,
		referenceData: b.referenceData,
	}
}
