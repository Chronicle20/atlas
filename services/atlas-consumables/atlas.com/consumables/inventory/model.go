package inventory

import (
	"atlas-consumables/compartment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func BuilderSupplier(characterId uint32) model.Provider[*Builder] {
	return func() (*Builder, error) {
		return NewBuilder(characterId), nil
	}
}

func FoldCompartment(b *Builder, m compartment.Model) (*Builder, error) {
	return b.SetCompartment(m), nil
}
