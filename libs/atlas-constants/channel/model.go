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

func NewModel(worldId world.Id, id Id) Model {
	return Model{
		worldId: worldId,
		id:      id,
	}
}
