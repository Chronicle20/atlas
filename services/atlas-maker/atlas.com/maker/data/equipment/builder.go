package equipment

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// Builder constructs a Model.
type Builder struct {
	id       item.Id
	reqLevel uint32
}

func NewBuilder(id item.Id) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetReqLevel(reqLevel uint32) *Builder {
	b.reqLevel = reqLevel
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		id:       b.id,
		reqLevel: b.reqLevel,
	}, nil
}
