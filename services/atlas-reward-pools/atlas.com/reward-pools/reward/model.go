package reward

type Model struct {
	itemId      uint32
	quantity    uint32
	tier        string
	weight      uint32
	gachaponId  string
	commodityId uint32
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Tier() string {
	return m.tier
}

func (m Model) Weight() uint32 {
	return m.weight
}

func (m Model) GachaponId() string {
	return m.gachaponId
}

// CommodityId is the cash shop commodity (serial number) this reward grants.
// Non-zero only for cash-surprise pools; other kinds identify the reward by
// ItemId alone.
func (m Model) CommodityId() uint32 {
	return m.commodityId
}
