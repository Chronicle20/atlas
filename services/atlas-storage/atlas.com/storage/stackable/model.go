package stackable

// Model represents stackable item data (consumable, setup, etc)
type Model struct {
	assetId  uint32
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func (m Model) AssetId() uint32 {
	return m.assetId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) Flag() uint16 {
	return m.flag
}
