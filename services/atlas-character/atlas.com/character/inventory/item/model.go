package item

type Model struct {
	id       uint32
	itemId   uint32
	slot     int16
	quantity uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func Id(m Model) (uint32, error) {
	return m.Id(), nil
}

type ModelBuilder struct {
	model Model
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func (b *ModelBuilder) SetID(id uint32) *ModelBuilder {
	b.model.id = id
	return b
}

func (b *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	b.model.itemId = itemId
	return b
}

func (b *ModelBuilder) SetSlot(slot int16) *ModelBuilder {
	b.model.slot = slot
	return b
}

func (b *ModelBuilder) SetQuantity(quantity uint32) *ModelBuilder {
	b.model.quantity = quantity
	return b
}

func (b *ModelBuilder) Build() Model {
	return b.model
}
