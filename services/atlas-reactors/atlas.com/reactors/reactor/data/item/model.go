package item

type Model struct {
	itemId   uint32
	quantity uint16
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint16 {
	return m.quantity
}
