package inventory

import (
	"atlas-channel/compartment"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

var ErrInvalidCharacterId = errors.New("character id must be greater than 0")

type builder struct {
	characterId  uint32
	compartments map[inventory.Type]compartment.Model
}

// NewBuilder creates a new builder instance
func NewBuilder(characterId uint32) *builder {
	return &builder{
		characterId:  characterId,
		compartments: make(map[inventory.Type]compartment.Model),
	}
}

// BuilderSupplier returns a provider for a new builder
func BuilderSupplier(characterId uint32) model.Provider[*builder] {
	return func() (*builder, error) {
		return NewBuilder(characterId), nil
	}
}

// FoldCompartment adds a compartment to the builder
func FoldCompartment(b *builder, m compartment.Model) (*builder, error) {
	return b.SetCompartment(m), nil
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *builder {
	return &builder{
		characterId:  m.characterId,
		compartments: m.compartments,
	}
}

// SetCompartment sets a compartment by its type
func (b *builder) SetCompartment(m compartment.Model) *builder {
	b.compartments[m.Type()] = m
	return b
}

// SetEquipable sets the equip compartment
func (b *builder) SetEquipable(m compartment.Model) *builder {
	b.compartments[inventory.TypeValueEquip] = m
	return b
}

// SetConsumable sets the use compartment
func (b *builder) SetConsumable(m compartment.Model) *builder {
	b.compartments[inventory.TypeValueUse] = m
	return b
}

// SetSetup sets the setup compartment
func (b *builder) SetSetup(m compartment.Model) *builder {
	b.compartments[inventory.TypeValueSetup] = m
	return b
}

// SetEtc sets the ETC compartment
func (b *builder) SetEtc(m compartment.Model) *builder {
	b.compartments[inventory.TypeValueETC] = m
	return b
}

// SetCash sets the cash compartment
func (b *builder) SetCash(m compartment.Model) *builder {
	b.compartments[inventory.TypeValueCash] = m
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, ErrInvalidCharacterId
	}
	return Model{
		characterId:  b.characterId,
		compartments: b.compartments,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
