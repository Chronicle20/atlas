package reward

type Builder struct {
	itemId     uint32
	quantity   uint32
	tier       string
	gachaponId string
}

func NewBuilder(gachaponId string) *Builder {
	return &Builder{gachaponId: gachaponId}
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetQuantity(quantity uint32) *Builder {
	b.quantity = quantity
	return b
}

func (b *Builder) SetTier(tier string) *Builder {
	b.tier = tier
	return b
}

func (b *Builder) Build() Model {
	return Model{
		itemId:     b.itemId,
		quantity:   b.quantity,
		tier:       b.tier,
		gachaponId: b.gachaponId,
	}
}
