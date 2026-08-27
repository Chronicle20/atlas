package inventory

import (
	"atlas-cashshop/cashshop/inventory/compartment"
)

// Clone creates a builder from this model
func Clone(m Model) *Builder {
	return &Builder{
		accountId:    m.accountId,
		compartments: m.compartments,
	}
}

// Builder is a builder for the Model
type Builder struct {
	accountId    uint32
	compartments map[compartment.CompartmentType]compartment.Model
}

// NewBuilder creates a new Builder
func NewBuilder(accountId uint32) *Builder {
	return &Builder{
		accountId:    accountId,
		compartments: make(map[compartment.CompartmentType]compartment.Model),
	}
}

// SetCompartment adds a compartment to the builder
func (b *Builder) SetCompartment(m compartment.Model) *Builder {
	b.compartments[m.Type()] = m
	return b
}

// SetExplorer sets the Explorer compartment
func (b *Builder) SetExplorer(m compartment.Model) *Builder {
	b.compartments[compartment.TypeExplorer] = m
	return b
}

// SetCygnus sets the Cygnus compartment
func (b *Builder) SetCygnus(m compartment.Model) *Builder {
	b.compartments[compartment.TypeCygnus] = m
	return b
}

// SetLegend sets the Legend compartment
func (b *Builder) SetLegend(m compartment.Model) *Builder {
	b.compartments[compartment.TypeLegend] = m
	return b
}

// Build creates a Model from this builder
func (b *Builder) Build() Model {
	return Model{
		accountId:    b.accountId,
		compartments: b.compartments,
	}
}
