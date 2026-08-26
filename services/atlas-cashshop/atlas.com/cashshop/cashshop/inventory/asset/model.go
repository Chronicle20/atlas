package asset

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id               uint32
	compartmentId    uuid.UUID
	cashId           int64
	templateId       uint32
	commodityId      uint32
	currency         uint32
	quantity         uint32
	flag             uint16
	petId            uint32
	purchasedBy      uint32
	expiration       time.Time
	createdAt        time.Time
	giftFrom         string
	giftMessage      string
	giftAcknowledged bool
	giftNoteSent     bool
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

// Currency is the wallet bucket this asset was purchased with -- see
// Entity.Currency's doc comment for the full 0-means-default-bucket
// convention.
func (m Model) Currency() uint32 {
	return m.currency
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

// GiftFrom is the sender's character name for a GIFT purchase (task-240 task
// 13); empty for every other asset. See Entity.GiftFrom's doc comment for
// the 13-character wire bound.
func (m Model) GiftFrom() string {
	return m.giftFrom
}

// GiftMessage is the sender's message for a GIFT purchase; empty for every
// other asset. See Entity.GiftMessage's doc comment for the 73-character
// wire bound.
func (m Model) GiftMessage() string {
	return m.giftMessage
}

// GiftAcknowledged reports whether the gift list carrying this asset has
// already been presented to the recipient via a LOAD_GIFT_SUCCESS announce
// (task-240 Defect H). See Entity.GiftAcknowledged's doc comment: this is
// NOT "the recipient clicked OK."
func (m Model) GiftAcknowledged() bool {
	return m.giftAcknowledged
}

// GiftNoteSent reports whether the gift-forward note for this asset has
// already been sent to the gifter (task-240 Defect I). See
// Entity.GiftNoteSent's doc comment: this is a SECOND, independent flag from
// GiftAcknowledged -- do not conflate the two.
func (m Model) GiftNoteSent() bool {
	return m.giftNoteSent
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:               m.id,
		compartmentId:    m.compartmentId,
		cashId:           m.cashId,
		templateId:       m.templateId,
		commodityId:      m.commodityId,
		currency:         m.currency,
		quantity:         m.quantity,
		flag:             m.flag,
		petId:            m.petId,
		purchasedBy:      m.purchasedBy,
		expiration:       m.expiration,
		createdAt:        m.createdAt,
		giftFrom:         m.giftFrom,
		giftMessage:      m.giftMessage,
		giftAcknowledged: m.giftAcknowledged,
		giftNoteSent:     m.giftNoteSent,
	}
}

type ModelBuilder struct {
	id               uint32
	compartmentId    uuid.UUID
	cashId           int64
	templateId       uint32
	commodityId      uint32
	currency         uint32
	quantity         uint32
	flag             uint16
	petId            uint32
	purchasedBy      uint32
	expiration       time.Time
	createdAt        time.Time
	giftFrom         string
	giftMessage      string
	giftAcknowledged bool
	giftNoteSent     bool
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

func (b *ModelBuilder) SetCurrency(currency uint32) *ModelBuilder {
	b.currency = currency
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

func (b *ModelBuilder) SetPetId(petId uint32) *ModelBuilder {
	b.petId = petId
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

func (b *ModelBuilder) SetGiftFrom(giftFrom string) *ModelBuilder {
	b.giftFrom = giftFrom
	return b
}

func (b *ModelBuilder) SetGiftMessage(giftMessage string) *ModelBuilder {
	b.giftMessage = giftMessage
	return b
}

// SetGiftAcknowledged sets whether the gift list carrying this asset has
// already been presented to the recipient (task-240 Defect H).
func (b *ModelBuilder) SetGiftAcknowledged(giftAcknowledged bool) *ModelBuilder {
	b.giftAcknowledged = giftAcknowledged
	return b
}

// SetGiftNoteSent sets whether the gift-forward note for this asset has
// already been sent to the gifter (task-240 Defect I).
func (b *ModelBuilder) SetGiftNoteSent(giftNoteSent bool) *ModelBuilder {
	b.giftNoteSent = giftNoteSent
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:               b.id,
		compartmentId:    b.compartmentId,
		cashId:           b.cashId,
		templateId:       b.templateId,
		commodityId:      b.commodityId,
		currency:         b.currency,
		quantity:         b.quantity,
		flag:             b.flag,
		petId:            b.petId,
		purchasedBy:      b.purchasedBy,
		expiration:       b.expiration,
		createdAt:        b.createdAt,
		giftFrom:         b.giftFrom,
		giftMessage:      b.giftMessage,
		giftAcknowledged: b.giftAcknowledged,
		giftNoteSent:     b.giftNoteSent,
	}
}
