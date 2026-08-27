package asset

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id            uint32
	compartmentId uuid.UUID
	cashId        int64
	templateId    uint32
	commodityId   uint32
	quantity      uint32
	flag          uint16
	petId         uint32
	purchasedBy   uint32
	expiration    time.Time
	createdAt     time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CompartmentId() uuid.UUID {
	return m.compartmentId
}

func (m Model) CashId() int64 {
	return m.cashId
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) CommodityId() uint32 {
	return m.commodityId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Flag() uint16 {
	return m.flag
}

func (m Model) PetId() uint32 {
	return m.petId
}

func (m Model) PurchasedBy() uint32 {
	return m.purchasedBy
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}
