package asset

import (
	"github.com/google/uuid"
	"time"
)

// ReferenceType constants for different item types
type ReferenceType string

const (
	ReferenceTypeEquipable     ReferenceType = "equipable"
	ReferenceTypeCashEquipable ReferenceType = "cash_equipable"
	ReferenceTypeConsumable    ReferenceType = "consumable"
	ReferenceTypeSetup         ReferenceType = "setup"
	ReferenceTypeEtc           ReferenceType = "etc"
	ReferenceTypeCash          ReferenceType = "cash"
	ReferenceTypePet           ReferenceType = "pet"
)

// InventoryType represents the inventory category for the asset
type InventoryType byte

const (
	InventoryTypeEquip InventoryType = 1
	InventoryTypeUse   InventoryType = 2
	InventoryTypeSetup InventoryType = 3
	InventoryTypeEtc   InventoryType = 4
	InventoryTypeCash  InventoryType = 5
)

// Model represents a storage asset with generic reference data
type Model[E any] struct {
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

func (m Model[E]) Id() uint32 {
	return m.id
}

func (m Model[E]) StorageId() uuid.UUID {
	return m.storageId
}

func (m Model[E]) InventoryType() InventoryType {
	return m.inventoryType
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

func (m Model[E]) ReferenceData() E {
	return m.referenceData
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

func (m Model[E]) IsStackable() bool {
	return m.referenceType == ReferenceTypeConsumable ||
		m.referenceType == ReferenceTypeSetup ||
		m.referenceType == ReferenceTypeEtc
}

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

func (b *ModelBuilder[E]) Build() Model[E] {
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
	}
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

// InventoryTypeFromTemplateId determines the inventory type from a template ID
func InventoryTypeFromTemplateId(templateId uint32) InventoryType {
	category := templateId / 1000000
	switch category {
	case 1:
		return InventoryTypeEquip
	case 2:
		return InventoryTypeUse
	case 3:
		return InventoryTypeSetup
	case 4:
		return InventoryTypeEtc
	case 5:
		return InventoryTypeCash
	default:
		return InventoryTypeEtc
	}
}
