package reward

type Model struct {
	itemId     uint32
	quantity   uint32
	tier       string
	gachaponId string
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

func (m Model) GachaponId() string {
	return m.gachaponId
}
