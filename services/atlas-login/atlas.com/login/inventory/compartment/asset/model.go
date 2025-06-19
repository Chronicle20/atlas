package asset

import (
	"github.com/google/uuid"
	"time"
)

type ReferenceType string

const (
	ReferenceTypeEquipable     = ReferenceType("equipable")
	ReferenceTypeCashEquipable = ReferenceType("cash-equipable")
	ReferenceTypeConsumable    = ReferenceType("consumable")
	ReferenceTypeSetup         = ReferenceType("setup")
	ReferenceTypeEtc           = ReferenceType("etc")
	ReferenceTypeCash          = ReferenceType("cash")
	ReferenceTypePet           = ReferenceType("pet")
)

type Model[E any] struct {
	id            uint32
	compartmentId uuid.UUID
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType ReferenceType
	referenceData E
}

func (m Model[E]) Id() uint32 {
	return m.id
}

func (m Model[E]) Slot() int16 {
	return m.slot
}

func (m Model[E]) TemplateId() uint32 {
	return m.templateId
}

func (m Model[E]) Expiration() time.Time {
	return m.expiration
}

func (m Model[E]) ReferenceId() uint32 {
	return m.referenceId
}

func (m Model[E]) ReferenceType() ReferenceType {
	return m.referenceType
}

type HasQuantity interface {
	Quantity() uint32
}

func (m Model[E]) Quantity() uint32 {
	if q, ok := any(m.referenceData).(HasQuantity); ok {
		return q.Quantity()
	}
	return 1
}

func (m Model[E]) HasQuantity() bool {
	_, ok := any(m.referenceData).(HasQuantity)
	return ok
}

func (m Model[E]) IsEquipable() bool {
	return m.referenceType == ReferenceTypeEquipable
}

func (m Model[E]) IsCashEquipable() bool {
	return m.referenceType == ReferenceTypeCashEquipable
}

func (m Model[E]) IsConsumable() bool {
	return m.referenceType == ReferenceTypeConsumable
}

func (m Model[E]) IsSetup() bool {
	return m.referenceType == ReferenceTypeSetup
}

func (m Model[E]) IsEtc() bool {
	return m.referenceType == ReferenceTypeEtc
}

func (m Model[E]) IsCash() bool {
	return m.referenceType == ReferenceTypeCash
}

func (m Model[E]) IsPet() bool {
	return m.referenceType == ReferenceTypePet
}

func (m Model[E]) ReferenceData() E {
	return m.referenceData
}

func (m Model[E]) CompartmentId() uuid.UUID {
	return m.compartmentId
}

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
