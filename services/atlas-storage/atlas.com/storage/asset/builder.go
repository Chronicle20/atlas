package asset

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ModelBuilder for constructing Model instances
type ModelBuilder[E any] struct {
	id            uint32
	storageId     uuid.UUID
	inventoryType InventoryType
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType ReferenceType
	referenceData E
}

func NewModelBuilder[E any]() *ModelBuilder[E] {
	return &ModelBuilder[E]{}
}

func (b *ModelBuilder[E]) SetId(id uint32) *ModelBuilder[E] {
	b.id = id
	return b
}

func (b *ModelBuilder[E]) SetStorageId(storageId uuid.UUID) *ModelBuilder[E] {
	b.storageId = storageId
	return b
}

func (b *ModelBuilder[E]) SetInventoryType(inventoryType InventoryType) *ModelBuilder[E] {
	b.inventoryType = inventoryType
	return b
}

func (b *ModelBuilder[E]) SetSlot(slot int16) *ModelBuilder[E] {
	b.slot = slot
	return b
}

func (b *ModelBuilder[E]) SetTemplateId(templateId uint32) *ModelBuilder[E] {
	b.templateId = templateId
	return b
}

func (b *ModelBuilder[E]) SetExpiration(expiration time.Time) *ModelBuilder[E] {
	b.expiration = expiration
	return b
}

func (b *ModelBuilder[E]) SetReferenceId(referenceId uint32) *ModelBuilder[E] {
	b.referenceId = referenceId
	return b
}

func (b *ModelBuilder[E]) SetReferenceType(referenceType ReferenceType) *ModelBuilder[E] {
	b.referenceType = referenceType
	return b
}

func (b *ModelBuilder[E]) SetReferenceData(referenceData E) *ModelBuilder[E] {
	b.referenceData = referenceData
	return b
}

func (b *ModelBuilder[E]) validate() error {
	if b.storageId == uuid.Nil {
		return errors.New("storage id is required")
	}
	if b.templateId == 0 {
		return errors.New("template id is required")
	}
	if b.referenceType == "" {
		return errors.New("reference type is required")
	}
	return nil
}

func (b *ModelBuilder[E]) Build() (Model[E], error) {
	if err := b.validate(); err != nil {
		return Model[E]{}, err
	}
	return Model[E]{
		id:            b.id,
		storageId:     b.storageId,
		inventoryType: b.inventoryType,
		slot:          b.slot,
		templateId:    b.templateId,
		expiration:    b.expiration,
		referenceId:   b.referenceId,
		referenceType: b.referenceType,
		referenceData: b.referenceData,
	}, nil
}

// MustBuild builds the model, panicking on validation error.
// Use only for trusted internal data (e.g., from database entities).
func (b *ModelBuilder[E]) MustBuild() Model[E] {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// Clone creates a copy of the Model with modifications
func Clone[E any](m Model[E]) *ModelBuilder[E] {
	return &ModelBuilder[E]{
		id:            m.id,
		storageId:     m.storageId,
		inventoryType: m.inventoryType,
		slot:          m.slot,
		templateId:    m.templateId,
		expiration:    m.expiration,
		referenceId:   m.referenceId,
		referenceType: m.referenceType,
		referenceData: m.referenceData,
	}
}
