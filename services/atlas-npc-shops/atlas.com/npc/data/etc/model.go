package etc

type Model struct {
	id        uint32
	price     uint32
	unitPrice float64
	slotMax   uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Price() uint32 {
	return m.price
}

func (m Model) UnitPrice() float64 {
	return m.unitPrice
}

func (m Model) SlotMax() uint32 {
	return m.slotMax
}
