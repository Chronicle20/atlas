package pet

import "time"

type Model struct {
	id         uint32
	cashId     int64
	templateId uint32
	name       string
	level      byte
	closeness  uint16
	fullness   byte
	expiration time.Time
	ownerId    uint32
	slot       int8
	flag       uint16
	purchaseBy uint32
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

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Closeness() uint16 {
	return m.closeness
}

func (m Model) Fullness() byte {
	return m.fullness
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) Lead() bool {
	return m.Slot() == 0
}

func (m Model) Slot() int8 {
	return m.slot
}

func (m Model) Flag() uint16 {
	return m.flag
}

func (m Model) PurchaseBy() uint32 {
	return m.purchaseBy
}
