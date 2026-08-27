package compartment

import (
	"atlas-channel/cashshop/inventory/asset"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (zero UUID)
var ErrInvalidId = errors.New("id must not be zero UUID")

// ErrInvalidAccountId is returned when the accountId is invalid (zero)
var ErrInvalidAccountId = errors.New("accountId must be greater than 0")

// modelBuilder is a builder for the Model
type builder struct {
	id        uuid.UUID
	accountId uint32
	type_     CompartmentType
	capacity  uint32
	assets    []asset.Model
}

// NewBuilder creates a new builder with required fields
func NewBuilder(id uuid.UUID, accountId uint32, type_ CompartmentType, capacity uint32) *builder {
	return &builder{
		id:        id,
		accountId: accountId,
		type_:     type_,
		capacity:  capacity,
		assets:    make([]asset.Model, 0),
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *builder {
	return &builder{
		id:        m.id,
		accountId: m.accountId,
		type_:     m.type_,
		capacity:  m.capacity,
		assets:    m.assets,
	}
}

// SetId sets the id for the modelBuilder
func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

// SetAccountId sets the accountId for the modelBuilder
func (b *builder) SetAccountId(accountId uint32) *builder {
	b.accountId = accountId
	return b
}

// SetType sets the type for the modelBuilder
func (b *builder) SetType(type_ CompartmentType) *builder {
	b.type_ = type_
	return b
}

// SetCapacity sets the capacity of this compartment
func (b *builder) SetCapacity(capacity uint32) *builder {
	b.capacity = capacity
	return b
}

// AddAsset adds an asset to this compartment
func (b *builder) AddAsset(a asset.Model) *builder {
	b.assets = append(b.assets, a)
	return b
}

// SetAssets sets all assets in this compartment
func (b *builder) SetAssets(as []asset.Model) *builder {
	b.assets = as
	return b
}

// Build creates a Model from this builder
func (b *builder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrInvalidId
	}
	if b.accountId == 0 {
		return Model{}, ErrInvalidAccountId
	}
	return Model{
		id:        b.id,
		accountId: b.accountId,
		type_:     b.type_,
		capacity:  b.capacity,
		assets:    b.assets,
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
