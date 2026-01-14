package item

import "time"

type Model struct {
	id          uint32
	cashId      int64
	templateId  uint32
	quantity    uint32
	flag        uint16
	purchasedBy uint32
	expiration  time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CashId() int64 {
	return m.cashId
}

func (m Model) TemplateId() uint32 {
	return m.templateId
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
