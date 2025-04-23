package item

import "time"

type Model struct {
	id              uint64
	itemId          uint32
	serialNumber    uint32
	quantity        int16
	expiration      time.Time
	purchasedByName string
	paybackRate     uint32
	discountRate    uint32
}

func NewModel(id uint64, itemId uint32, serialNumber uint32, quantity int16) Model {
	return Model{
		id:              id,
		itemId:          itemId,
		serialNumber:    serialNumber,
		quantity:        quantity,
		expiration:      time.Now(),
		purchasedByName: "Atlas",
		paybackRate:     100,
		discountRate:    2,
	}
}

func (m Model) Id() uint64 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) SerialNumber() uint32 {
	return m.serialNumber
}

func (m Model) Quantity() int16 {
	return m.quantity
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) PurchasedByName() string {
	return m.purchasedByName
}

func (m Model) PaybackRate() uint32 {
	return m.paybackRate
}

func (m Model) DiscountRate() uint32 {
	return m.discountRate
}
