package inventory

import "atlas-channel/cashshop/inventory/item"

type Model struct {
	items []item.Model
}

func NewModel(items []item.Model) Model {
	return Model{
		items: items,
	}
}

func (m Model) Items() []item.Model {
	return m.items
}
