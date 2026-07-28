package searchcount

type Model struct {
	itemId uint32
	count  uint64
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Count() uint64 {
	return m.count
}
