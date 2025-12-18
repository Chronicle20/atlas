package commodity

type Model struct {
	id       uint32
	itemId   uint32
	count    uint32
	price    uint32
	period   uint32
	priority uint32
	gender   byte
	onSale   bool
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Price() uint32 {
	return m.price
}

func (m Model) Count() uint32 {
	return m.count
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:       rm.Id,
		itemId:   rm.ItemId,
		count:    rm.Count,
		price:    rm.Price,
		period:   rm.Period,
		priority: rm.Priority,
		gender:   rm.Gender,
		onSale:   rm.OnSale,
	}, nil
}
