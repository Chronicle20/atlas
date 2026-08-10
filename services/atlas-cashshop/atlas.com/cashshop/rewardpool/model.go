package rewardpool

type Model struct {
	itemId      uint32
	quantity    uint32
	commodityId uint32
}

func (m Model) ItemId() uint32      { return m.itemId }
func (m Model) Quantity() uint32    { return m.quantity }
func (m Model) CommodityId() uint32 { return m.commodityId }
