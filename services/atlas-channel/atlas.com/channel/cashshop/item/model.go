package item

import "time"

type Model struct {
	id          uint32
	cashId      uint64
	templateId  uint32
	quantity    uint32
	flag        uint16
	purchasedBy uint32
	expiration  time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CashId() uint64 {
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

type Builder struct {
	id          uint32
	cashId      uint64
	templateId  uint32
	quantity    uint32
	flag        uint16
	purchasedBy uint32
	expiration  time.Time
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetCashId(cashId uint64) *Builder {
	b.cashId = cashId
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
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

func (b *Builder) SetPurchasedBy(purchasedBy uint32) *Builder {
	b.purchasedBy = purchasedBy
	return b
}

func (b *Builder) SetExpiration(expiration time.Time) *Builder {
	b.expiration = expiration
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:          b.id,
		cashId:      b.cashId,
		templateId:  b.templateId,
		quantity:    b.quantity,
		flag:        b.flag,
		purchasedBy: b.purchasedBy,
		expiration:  b.expiration,
	}
}
