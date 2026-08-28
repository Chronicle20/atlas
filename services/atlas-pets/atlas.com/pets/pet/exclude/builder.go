package exclude

import "errors"

type Builder struct {
	id     uint32
	itemId uint32
}

func NewBuilder(itemId uint32) *Builder {
	return &Builder{
		itemId: itemId,
	}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.itemId == 0 {
		return Model{}, errors.New("itemId is required")
	}
	return Model{
		id:     b.id,
		itemId: b.itemId,
	}, nil
}
