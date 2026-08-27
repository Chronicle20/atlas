package inventory

import (
	"atlas-channel/cashshop/inventory/compartment"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ErrInvalidAccountId is returned when the accountId is invalid (zero)
var ErrInvalidAccountId = errors.New("accountId must be greater than 0")

// modelBuilder is a builder for the Model
type builder struct {
	accountId    uint32
	compartments map[compartment.CompartmentType]compartment.Model
}

// NewBuilder creates a new builder with required accountId
func NewBuilder(accountId uint32) *builder {
	return &builder{
		accountId:    accountId,
		compartments: make(map[compartment.CompartmentType]compartment.Model),
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *builder {
	// Deep copy the compartments map to avoid shared reference
	compartments := make(map[compartment.CompartmentType]compartment.Model)
	for k, v := range m.compartments {
		compartments[k] = v
	}
	return &builder{
		accountId:    m.accountId,
		compartments: compartments,
	}
}

// BuilderSupplier provides a new modelBuilder for folding operations
func BuilderSupplier(accountId uint32) model.Provider[*builder] {
	return func() (*builder, error) {
		return NewBuilder(accountId), nil
	}
}

// FoldCompartment adds a compartment to the builder (for use with model.Fold)
func FoldCompartment(b *builder, m compartment.Model) (*builder, error) {
	return b.SetCompartment(m), nil
}

// SetAccountId sets the accountId for the modelBuilder
func (b *builder) SetAccountId(accountId uint32) *builder {
	b.accountId = accountId
	return b
}

// SetCompartment adds a compartment to the builder
func (b *builder) SetCompartment(m compartment.Model) *builder {
	b.compartments[m.Type()] = m
	return b
}

// SetExplorer sets the Explorer compartment
func (b *builder) SetExplorer(m compartment.Model) *builder {
	b.compartments[compartment.TypeExplorer] = m
	return b
}

// SetCygnus sets the Cygnus compartment
func (b *builder) SetCygnus(m compartment.Model) *builder {
	b.compartments[compartment.TypeCygnus] = m
	return b
}

// SetLegend sets the Legend compartment
func (b *builder) SetLegend(m compartment.Model) *builder {
	b.compartments[compartment.TypeLegend] = m
	return b
}

// Build creates a Model from this builder
func (b *builder) Build() (Model, error) {
	if b.accountId == 0 {
		return Model{}, ErrInvalidAccountId
	}
	return Model{
		accountId:    b.accountId,
		compartments: b.compartments,
	}, nil
}

// MustBuild creates a Model from this builder and panics if validation fails
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
