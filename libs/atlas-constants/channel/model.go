package channel

import "github.com/Chronicle20/atlas-constants/world"

type Model struct {
	worldId world.Id
	id      Id
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) Id() Id {
	return m.id
}

func (m Model) Clone() *Builder {
	return NewBuilder(m.WorldId(), m.Id())
}

func NewModel(worldId world.Id, id Id) Model {
	return Model{
		worldId: worldId,
		id:      id,
	}
}

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
