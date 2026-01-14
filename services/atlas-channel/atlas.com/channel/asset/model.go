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

// InventoryType represents the inventory category for the asset
type InventoryType byte

const (
	InventoryTypeEquip InventoryType = 1
	InventoryTypeUse   InventoryType = 2
	InventoryTypeSetup InventoryType = 3
	InventoryTypeEtc   InventoryType = 4
	InventoryTypeCash  InventoryType = 5
)

type Model[E any] struct {
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

func (m Model[E]) Id() uint32 {
	return m.id
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

