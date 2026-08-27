package inventory

import (
	"atlas-pets/compartment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

func Clone(m Model) *Builder {
	return &Builder{
		characterId:  m.characterId,
		compartments: m.compartments,
	}
}

type Builder struct {
	characterId  uint32
	compartments map[inventory.Type]compartment.Model
}

func NewBuilder(characterId uint32) *Builder {
	return &Builder{
		characterId:  characterId,
		compartments: make(map[inventory.Type]compartment.Model),
	}
}

func (b *Builder) SetCompartment(m compartment.Model) *Builder {
	b.compartments[m.Type()] = m
	return b
}

func (b *Builder) SetEquipable(m compartment.Model) *Builder {
	b.compartments[inventory.TypeValueEquip] = m
	return b
}

func (b *Builder) SetConsumable(m compartment.Model) *Builder {
	b.compartments[inventory.TypeValueUse] = m
	return b
}

func (b *Builder) SetSetup(m compartment.Model) *Builder {
	b.compartments[inventory.TypeValueSetup] = m
	return b
}

func (b *Builder) SetEtc(m compartment.Model) *Builder {
	b.compartments[inventory.TypeValueETC] = m
	return b
}

func (b *Builder) SetCash(m compartment.Model) *Builder {
	b.compartments[inventory.TypeValueCash] = m
	return b
}

func (b *Builder) Build() Model {
	return Model{
		characterId:  b.characterId,
		compartments: b.compartments,
	}
}
