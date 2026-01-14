package inventory

import (
	"atlas-channel/cashshop/inventory/compartment"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
)

// ErrInvalidAccountId is returned when the accountId is invalid (zero)
var ErrInvalidAccountId = errors.New("accountId must be greater than 0")

// modelBuilder is a builder for the Model
type modelBuilder struct {
	accountId    uint32
	compartments map[compartment.CompartmentType]compartment.Model
}

// NewModelBuilder creates a new modelBuilder with required accountId
func NewModelBuilder(accountId uint32) *modelBuilder {
	return &modelBuilder{
		accountId:    accountId,
		compartments: make(map[compartment.CompartmentType]compartment.Model),
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *modelBuilder {
	// Deep copy the compartments map to avoid shared reference
	compartments := make(map[compartment.CompartmentType]compartment.Model)
	for k, v := range m.compartments {
		compartments[k] = v
	}
	return &modelBuilder{
		accountId:    m.accountId,
		compartments: compartments,
	}
}

// BuilderSupplier provides a new modelBuilder for folding operations
func BuilderSupplier(accountId uint32) model.Provider[*modelBuilder] {
	return func() (*modelBuilder, error) {
		return NewModelBuilder(accountId), nil
	}
}

// FoldCompartment adds a compartment to the builder (for use with model.Fold)
func FoldCompartment(b *modelBuilder, m compartment.Model) (*modelBuilder, error) {
	return b.SetCompartment(m), nil
}

// SetAccountId sets the accountId for the modelBuilder
func (b *modelBuilder) SetAccountId(accountId uint32) *modelBuilder {
	b.accountId = accountId
	return b
}

// SetCompartment adds a compartment to the builder
func (b *modelBuilder) SetCompartment(m compartment.Model) *modelBuilder {
	b.compartments[m.Type()] = m
	return b
}

// SetExplorer sets the Explorer compartment
func (b *modelBuilder) SetExplorer(m compartment.Model) *modelBuilder {
	b.compartments[compartment.TypeExplorer] = m
	return b
}

// SetCygnus sets the Cygnus compartment
func (b *modelBuilder) SetCygnus(m compartment.Model) *modelBuilder {
	b.compartments[compartment.TypeCygnus] = m
	return b
}

// SetLegend sets the Legend compartment
func (b *modelBuilder) SetLegend(m compartment.Model) *modelBuilder {
	b.compartments[compartment.TypeLegend] = m
	return b
}

// Build creates a Model from this builder
func (b *modelBuilder) Build() (Model, error) {
	if b.accountId == 0 {
		return Model{}, ErrInvalidAccountId
	}
	return Model{
		accountId:    b.accountId,
		compartments: b.compartments,
	}, nil
}

// MustBuild creates a Model from this builder and panics if validation fails
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
