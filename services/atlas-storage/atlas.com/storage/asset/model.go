package asset

import (
	"time"

	"github.com/google/uuid"
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
