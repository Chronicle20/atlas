package exclude

import "errors"

type ModelBuilder struct {
	id     uint32
	itemId uint32
}

func NewModelBuilder(itemId uint32) *ModelBuilder {
	return &ModelBuilder{
		itemId: itemId,
	}
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) Build() (Model, error) {
	if b.itemId == 0 {
		return Model{}, errors.New("itemId is required")
	}
	return Model{
		id:     b.id,
		itemId: b.itemId,
	}, nil
}
