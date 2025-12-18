package inventory

import (
	"atlas-cashshop/character/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
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

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		characterId:  m.characterId,
		compartments: m.compartments,
	}
}

type ModelBuilder struct {
	characterId  uint32
	compartments map[inventory.Type]compartment.Model
}

func NewBuilder(characterId uint32) *ModelBuilder {
	return &ModelBuilder{
		characterId:  characterId,
		compartments: make(map[inventory.Type]compartment.Model),
	}
}

func BuilderSupplier(characterId uint32) model.Provider[*ModelBuilder] {
	return func() (*ModelBuilder, error) {
		return NewBuilder(characterId), nil
	}
}

func FoldCompartment(b *ModelBuilder, m compartment.Model) (*ModelBuilder, error) {
	return b.SetCompartment(m), nil
}

func (b *ModelBuilder) SetCompartment(m compartment.Model) *ModelBuilder {
	b.compartments[m.Type()] = m
	return b
}

func (b *ModelBuilder) SetEquipable(m compartment.Model) *ModelBuilder {
	b.compartments[inventory.TypeValueEquip] = m
	return b
}

func (b *ModelBuilder) SetConsumable(m compartment.Model) *ModelBuilder {
	b.compartments[inventory.TypeValueUse] = m
	return b
}

func (b *ModelBuilder) SetSetup(m compartment.Model) *ModelBuilder {
	b.compartments[inventory.TypeValueSetup] = m
	return b
}

func (b *ModelBuilder) SetEtc(m compartment.Model) *ModelBuilder {
	b.compartments[inventory.TypeValueETC] = m
	return b
}

func (b *ModelBuilder) SetCash(m compartment.Model) *ModelBuilder {
	b.compartments[inventory.TypeValueCash] = m
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		characterId:  b.characterId,
		compartments: b.compartments,
	}
}
