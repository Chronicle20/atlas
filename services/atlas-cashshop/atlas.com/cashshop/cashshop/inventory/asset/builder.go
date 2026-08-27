package asset

import (
	"time"

	"github.com/google/uuid"
)

func Clone(m Model) *Builder {
	return &Builder{
		id:            m.id,
		compartmentId: m.compartmentId,
		cashId:        m.cashId,
		templateId:    m.templateId,
		commodityId:   m.commodityId,
		quantity:      m.quantity,
		flag:          m.flag,
		petId:         m.petId,
		purchasedBy:   m.purchasedBy,
		expiration:    m.expiration,
		createdAt:     m.createdAt,
	}
}

type Builder struct {
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

func NewBuilder(compartmentId uuid.UUID, templateId uint32) *Builder {
	return &Builder{
		compartmentId: compartmentId,
		templateId:    templateId,
	}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetCompartmentId(compartmentId uuid.UUID) *Builder {
	b.compartmentId = compartmentId
	return b
}

func (b *Builder) SetCashId(cashId int64) *Builder {
	b.cashId = cashId
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

func (b *Builder) SetCommodityId(commodityId uint32) *Builder {
	b.commodityId = commodityId
	return b
}

func (b *Builder) SetQuantity(quantity uint32) *Builder {
	b.quantity = quantity
	return b
}

func (b *Builder) SetFlag(flag uint16) *Builder {
	b.flag = flag
	return b
}

func (b *Builder) SetPetId(petId uint32) *Builder {
	b.petId = petId
	return b
}

func (b *Builder) SetPurchasedBy(purchasedBy uint32) *Builder {
	b.purchasedBy = purchasedBy
	return b
}

func (b *Builder) SetExpiration(expiration time.Time) *Builder {
	b.expiration = expiration
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:            b.id,
		compartmentId: b.compartmentId,
		cashId:        b.cashId,
		templateId:    b.templateId,
		commodityId:   b.commodityId,
		quantity:      b.quantity,
		flag:          b.flag,
		petId:         b.petId,
		purchasedBy:   b.purchasedBy,
		expiration:    b.expiration,
		createdAt:     b.createdAt,
	}
}
