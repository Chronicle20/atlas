package compartment

import (
	"atlas-inventory/asset"
	"errors"
	"sort"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

type Model struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model[any]
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

func (m Model) Assets() []asset.Model[any] {
	return m.assets
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) NextFreeSlot() (int16, error) {
	if len(m.Assets()) == 0 {
		return 1, nil
	}
	sort.Slice(m.Assets(), func(i, j int) bool {
		return m.Assets()[i].Slot() < m.Assets()[j].Slot()
	})

	slot := int16(1)
	i := 0

	for {
		if slot > int16(m.Capacity()) {
			return 0, errors.New("no free slots")
		} else if i >= len(m.Assets()) {
			return slot, nil
		} else if slot < m.Assets()[i].Slot() {
			return slot, nil
		} else if slot == m.Assets()[i].Slot() {
			slot += 1
			i += 1
		} else if m.Assets()[i].Slot() <= 0 {
			i += 1
		}
	}
}
