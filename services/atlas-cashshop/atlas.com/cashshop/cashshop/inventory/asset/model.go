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

func (m Model) PurchasedBy() uint32 {
	return m.purchasedBy
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:            m.id,
		compartmentId: m.compartmentId,
		cashId:        m.cashId,
		templateId:    m.templateId,
		commodityId:   m.commodityId,
		quantity:      m.quantity,
		flag:          m.flag,
		purchasedBy:   m.purchasedBy,
		expiration:    m.expiration,
		createdAt:     m.createdAt,
	}
}

type ModelBuilder struct {
	id            uint32
	compartmentId uuid.UUID
	cashId        int64
	templateId    uint32
	commodityId   uint32
	quantity      uint32
	flag          uint16
	purchasedBy   uint32
	expiration    time.Time
	createdAt     time.Time
}

func NewBuilder(compartmentId uuid.UUID, templateId uint32) *ModelBuilder {
	return &ModelBuilder{
		compartmentId: compartmentId,
		templateId:    templateId,
	}
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetCompartmentId(compartmentId uuid.UUID) *ModelBuilder {
	b.compartmentId = compartmentId
	return b
}

func (b *ModelBuilder) SetCashId(cashId int64) *ModelBuilder {
	b.cashId = cashId
	return b
}

func (b *ModelBuilder) SetTemplateId(templateId uint32) *ModelBuilder {
	b.templateId = templateId
	return b
}

func (b *ModelBuilder) SetCommodityId(commodityId uint32) *ModelBuilder {
	b.commodityId = commodityId
	return b
}

func (b *ModelBuilder) SetQuantity(quantity uint32) *ModelBuilder {
	b.quantity = quantity
	return b
}

func (b *ModelBuilder) SetFlag(flag uint16) *ModelBuilder {
	b.flag = flag
	return b
}

func (b *ModelBuilder) SetPurchasedBy(purchasedBy uint32) *ModelBuilder {
	b.purchasedBy = purchasedBy
	return b
}

func (b *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	b.expiration = expiration
	return b
}

func (b *ModelBuilder) SetCreatedAt(createdAt time.Time) *ModelBuilder {
	b.createdAt = createdAt
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:            b.id,
		compartmentId: b.compartmentId,
		cashId:        b.cashId,
		templateId:    b.templateId,
		commodityId:   b.commodityId,
		quantity:      b.quantity,
		flag:          b.flag,
		purchasedBy:   b.purchasedBy,
		expiration:    b.expiration,
		createdAt:     b.createdAt,
	}
}
