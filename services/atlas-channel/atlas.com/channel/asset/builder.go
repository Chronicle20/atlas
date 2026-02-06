package asset

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidId = errors.New("asset id must be greater than 0")
)

type modelBuilder[E any] struct {
	id            uint32
	compartmentId uuid.UUID
	inventoryType InventoryType
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType ReferenceType
	referenceData E
}

func NewModelBuilder[E any](id uint32, compartmentId uuid.UUID, templateId uint32, referenceId uint32, referenceType ReferenceType) *modelBuilder[E] {
	return &modelBuilder[E]{
		id:            id,
		compartmentId: compartmentId,
		slot:          0,
		templateId:    templateId,
		expiration:    time.Time{},
		referenceId:   referenceId,
		referenceType: referenceType,
	}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder[E any](id uint32, compartmentId uuid.UUID, templateId uint32, referenceId uint32, referenceType ReferenceType) *modelBuilder[E] {
	return NewModelBuilder[E](id, compartmentId, templateId, referenceId, referenceType)
}

func CloneModel[E any](m Model[E]) *modelBuilder[E] {
	return &modelBuilder[E]{
		id:            m.id,
		compartmentId: m.compartmentId,
		inventoryType: m.inventoryType,
		slot:          m.slot,
		templateId:    m.templateId,
		expiration:    m.expiration,
		referenceId:   m.referenceId,
		referenceType: m.referenceType,
		referenceData: m.referenceData,
	}
}

// Clone is an alias for CloneModel for backward compatibility
func Clone[E any](m Model[E]) *modelBuilder[E] {
	return CloneModel[E](m)
}

func (b *modelBuilder[E]) SetInventoryType(inventoryType InventoryType) *modelBuilder[E] {
	b.inventoryType = inventoryType
	return b
}

func (b *modelBuilder[E]) SetSlot(slot int16) *modelBuilder[E] {
	b.slot = slot
	return b
}

func (b *modelBuilder[E]) SetExpiration(e time.Time) *modelBuilder[E] {
	b.expiration = e
	return b
}

func (b *modelBuilder[E]) SetReferenceData(e E) *modelBuilder[E] {
	b.referenceData = e
	return b
}

func (b *modelBuilder[E]) Build() (Model[E], error) {
	if b.id == 0 {
		return Model[E]{}, ErrInvalidId
	}
	return Model[E]{
		id:            b.id,
		compartmentId: b.compartmentId,
		inventoryType: b.inventoryType,
		slot:          b.slot,
		templateId:    b.templateId,
		expiration:    b.expiration,
		referenceId:   b.referenceId,
		referenceType: b.referenceType,
		referenceData: b.referenceData,
	}, nil
}

func (b *modelBuilder[E]) MustBuild() Model[E] {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
