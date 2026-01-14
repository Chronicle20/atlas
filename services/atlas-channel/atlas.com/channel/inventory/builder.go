package inventory

import (
	"atlas-channel/compartment"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
)

var (
	ErrInvalidCharacterId = errors.New("character id must be greater than 0")
)

type modelBuilder struct {
	characterId  uint32
	compartments map[inventory.Type]compartment.Model
}

// NewModelBuilder creates a new builder instance
func NewModelBuilder(characterId uint32) *modelBuilder {
	return &modelBuilder{
		characterId:  characterId,
		compartments: make(map[inventory.Type]compartment.Model),
	}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder(characterId uint32) *modelBuilder {
	return NewModelBuilder(characterId)
}

// BuilderSupplier returns a provider for a new builder
func BuilderSupplier(characterId uint32) model.Provider[*modelBuilder] {
	return func() (*modelBuilder, error) {
		return NewBuilder(characterId), nil
	}
}

// FoldCompartment adds a compartment to the builder
func FoldCompartment(b *modelBuilder, m compartment.Model) (*modelBuilder, error) {
	return b.SetCompartment(m), nil
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		characterId:  m.characterId,
		compartments: m.compartments,
	}
}

// SetCompartment sets a compartment by its type
func (b *modelBuilder) SetCompartment(m compartment.Model) *modelBuilder {
	b.compartments[m.Type()] = m
	return b
}

// SetEquipable sets the equip compartment
func (b *modelBuilder) SetEquipable(m compartment.Model) *modelBuilder {
	b.compartments[inventory.TypeValueEquip] = m
	return b
}

// SetConsumable sets the use compartment
func (b *modelBuilder) SetConsumable(m compartment.Model) *modelBuilder {
	b.compartments[inventory.TypeValueUse] = m
	return b
}

// SetSetup sets the setup compartment
func (b *modelBuilder) SetSetup(m compartment.Model) *modelBuilder {
	b.compartments[inventory.TypeValueSetup] = m
	return b
}

// SetEtc sets the ETC compartment
func (b *modelBuilder) SetEtc(m compartment.Model) *modelBuilder {
	b.compartments[inventory.TypeValueETC] = m
	return b
}

// SetCash sets the cash compartment
func (b *modelBuilder) SetCash(m compartment.Model) *modelBuilder {
	b.compartments[inventory.TypeValueCash] = m
	return b
}

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, ErrInvalidCharacterId
	}
	return Model{
		characterId:  b.characterId,
		compartments: b.compartments,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
