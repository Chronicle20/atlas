package etc

type Model struct {
	id        uint32
	price     uint32
	unitPrice uint32
	slotMax   uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Price() uint32 {
	return m.price
}

func (m Model) UnitPrice() uint32 {
	return m.unitPrice
}

func (m Model) SlotMax() uint32 {
	return m.slotMax
}