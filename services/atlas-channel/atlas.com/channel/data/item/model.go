package item

// Model is one item-string search hit: the item's template id and its name.
type Model struct {
	itemId uint32
	name   string
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Name() string {
	return m.name
}
