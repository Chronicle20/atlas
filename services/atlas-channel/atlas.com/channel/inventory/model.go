package inventory

import (
	"atlas-channel/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

type Model struct {
	characterId  uint32
	compartments map[inventory.Type]compartment.Model
}

func (m Model) Equipable() compartment.Model {
	return m.compartments[inventory.TypeValueEquip]
}

func (m Model) Consumable() compartment.Model {
	return m.compartments[inventory.TypeValueUse]
}

func (m Model) Setup() compartment.Model {
	return m.compartments[inventory.TypeValueSetup]
}

func (m Model) ETC() compartment.Model {
	return m.compartments[inventory.TypeValueETC]
}

func (m Model) Cash() compartment.Model {
	return m.compartments[inventory.TypeValueCash]
}

func (m Model) CompartmentByType(it inventory.Type) compartment.Model {
	return m.compartments[it]
}

func (m Model) CompartmentById(id uuid.UUID) (compartment.Model, bool) {
	for _, c := range m.compartments {
		if c.Id() == id {
			return c, true
		}
	}
	return compartment.Model{}, false
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Compartments() []compartment.Model {
	res := make([]compartment.Model, 0)
	for _, v := range m.compartments {
		res = append(res, v)
	}
	return res
}

