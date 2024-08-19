package item

type Model struct {
	id       uint32
	itemId   uint32
	slot     int16
	quantity uint32
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Expiration() int64 {
	return -1
}

func (m Model) Owner() string {
	return ""
}

func (m Model) Flag() uint16 {
	return 0
}
