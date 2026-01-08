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

// ModelBuilder for constructing Model instances
type ModelBuilder struct {
	assetId  uint32
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		quantity: 1,
	}
}

func (b *ModelBuilder) SetAssetId(assetId uint32) *ModelBuilder {
	b.assetId = assetId
	return b
}

func (b *ModelBuilder) SetQuantity(quantity uint32) *ModelBuilder {
	b.quantity = quantity
	return b
}

func (b *ModelBuilder) SetOwnerId(ownerId uint32) *ModelBuilder {
	b.ownerId = ownerId
	return b
}

func (b *ModelBuilder) SetFlag(flag uint16) *ModelBuilder {
	b.flag = flag
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		assetId:  b.assetId,
		quantity: b.quantity,
		ownerId:  b.ownerId,
		flag:     b.flag,
	}
}

// Clone creates a copy of the Model with modifications
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		assetId:  m.assetId,
		quantity: m.quantity,
		ownerId:  m.ownerId,
		flag:     m.flag,
	}
}
