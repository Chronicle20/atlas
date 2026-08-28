package compartment

import (
	"atlas-query-aggregator/asset"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

type Model struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) Type() inventory.Type {
	return m.inventoryType
}

func (m Model) Capacity() uint32 {
	return m.capacity
}

func (m Model) Assets() []asset.Model {
	return m.assets
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}
