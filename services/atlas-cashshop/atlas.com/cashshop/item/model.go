package item

type Model struct {
	id          uint32
	cashId      uint64
	templateId  uint32
	quantity    uint32
	owner       uint32
	flag        uint16
	purchasedBy uint32
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

func (m Model) Owner() uint32 {
	return m.owner
}

func (m Model) Flag() uint16 {
	return m.flag
}

func (m Model) PurchasedBy() uint32 {
	return m.purchasedBy
}
