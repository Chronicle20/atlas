package item

type Model struct {
	id       uint32
	itemId   uint32
	slot     int16
	quantity uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) Quantity() uint32 {
	return m.quantity
}
