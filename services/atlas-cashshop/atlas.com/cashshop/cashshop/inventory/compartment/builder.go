package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"

	"github.com/google/uuid"
)

// Clone creates a builder from this model
func Clone(m Model) *Builder {
	return &Builder{
		id:        m.id,
		accountId: m.accountId,
		type_:     m.type_,
		capacity:  m.capacity,
		assets:    m.assets,
	}
}

// Builder is a builder for the Model
type Builder struct {
	id        uuid.UUID
	accountId uint32
	type_     CompartmentType
	capacity  uint32
	assets    []asset.Model
}

// NewBuilder creates a new Builder
func NewBuilder(id uuid.UUID, accountId uint32, type_ CompartmentType, capacity uint32) *Builder {
	return &Builder{
		id:        id,
		accountId: accountId,
		type_:     type_,
		capacity:  capacity,
		assets:    make([]asset.Model, 0),
	}
}

// SetCapacity sets the capacity of this compartment
func (b *Builder) SetCapacity(capacity uint32) *Builder {
	b.capacity = capacity
	return b
}

// AddAsset adds an asset to this compartment
func (b *Builder) AddAsset(a asset.Model) *Builder {
	b.assets = append(b.assets, a)
	return b
}

// SetAssets sets all assets in this compartment
func (b *Builder) SetAssets(as []asset.Model) *Builder {
	b.assets = as
	return b
}

// Build creates a Model from this builder
func (b *Builder) Build() Model {
	return Model{
		id:        b.id,
		accountId: b.accountId,
		type_:     b.type_,
		capacity:  b.capacity,
		assets:    b.assets,
	}
}
