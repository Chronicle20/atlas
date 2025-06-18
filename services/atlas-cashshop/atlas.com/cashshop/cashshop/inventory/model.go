package inventory

import (
	"atlas-cashshop/cashshop/inventory/compartment"
	"github.com/Chronicle20/atlas-model/model"
)

// Model represents a cash shop inventory with multiple compartments
type Model struct {
	accountId    uint32
	compartments map[compartment.CompartmentType]compartment.Model
}

// AccountId returns the account ID associated with this inventory
func (m Model) AccountId() uint32 {
	return m.accountId
}

// Compartments returns all compartments in this inventory
func (m Model) Compartments() []compartment.Model {
	res := make([]compartment.Model, 0)
	for _, v := range m.compartments {
		res = append(res, v)
	}
	return res
}

// CompartmentByType returns a specific compartment by its type
func (m Model) CompartmentByType(ct compartment.CompartmentType) compartment.Model {
	return m.compartments[ct]
}

// Explorer returns the Explorer compartment
func (m Model) Explorer() compartment.Model {
	return m.compartments[compartment.TypeExplorer]
}

// Cygnus returns the Cygnus compartment
func (m Model) Cygnus() compartment.Model {
	return m.compartments[compartment.TypeCygnus]
}

// Legend returns the Legend compartment
func (m Model) Legend() compartment.Model {
	return m.compartments[compartment.TypeLegend]
}

// Clone creates a builder from this model
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		accountId:    m.accountId,
		compartments: m.compartments,
	}
}

// ModelBuilder is a builder for the Model
type ModelBuilder struct {
	accountId    uint32
	compartments map[compartment.CompartmentType]compartment.Model
}

// NewBuilder creates a new ModelBuilder
func NewBuilder(accountId uint32) *ModelBuilder {
	return &ModelBuilder{
		accountId:    accountId,
		compartments: make(map[compartment.CompartmentType]compartment.Model),
	}
}

// BuilderSupplier provides a new ModelBuilder
func BuilderSupplier(accountId uint32) model.Provider[*ModelBuilder] {
	return func() (*ModelBuilder, error) {
		return NewBuilder(accountId), nil
	}
}

// FoldCompartment adds a compartment to the builder
func FoldCompartment(b *ModelBuilder, m compartment.Model) (*ModelBuilder, error) {
	return b.SetCompartment(m), nil
}

// SetCompartment adds a compartment to the builder
func (b *ModelBuilder) SetCompartment(m compartment.Model) *ModelBuilder {
	b.compartments[m.Type()] = m
	return b
}

// SetExplorer sets the Explorer compartment
func (b *ModelBuilder) SetExplorer(m compartment.Model) *ModelBuilder {
	b.compartments[compartment.TypeExplorer] = m
	return b
}

// SetCygnus sets the Cygnus compartment
func (b *ModelBuilder) SetCygnus(m compartment.Model) *ModelBuilder {
	b.compartments[compartment.TypeCygnus] = m
	return b
}

// SetLegend sets the Legend compartment
func (b *ModelBuilder) SetLegend(m compartment.Model) *ModelBuilder {
	b.compartments[compartment.TypeLegend] = m
	return b
}

// Build creates a Model from this builder
func (b *ModelBuilder) Build() Model {
	return Model{
		accountId:    b.accountId,
		compartments: b.compartments,
	}
}
