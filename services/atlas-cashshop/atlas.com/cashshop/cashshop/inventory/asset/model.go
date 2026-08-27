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
