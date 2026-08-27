package channel

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct {
	worldId world.Id
	id      Id
}

func NewBuilder(worldId world.Id, id Id) *Builder {
	return &Builder{
		worldId: worldId,
		id:      id,
	}
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetId(id Id) *Builder {
	b.id = id
	return b
}

func (b *Builder) Build() Model {
	return NewModel(b.worldId, b.id)
}
