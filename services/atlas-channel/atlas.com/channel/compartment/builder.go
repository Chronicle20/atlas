package compartment

import (
	"atlas-channel/asset"
	"errors"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

var ErrMissingId = errors.New("compartment id is required")

type builder struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model
}

// NewBuilder creates a new builder instance with required fields
func NewBuilder(id uuid.UUID, characterId uint32, it inventory.Type, capacity uint32) *builder {
	return &builder{
		id:            id,
		characterId:   characterId,
		inventoryType: it,
		capacity:      capacity,
		assets:        make([]asset.Model, 0),
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *builder {
	return &builder{
		id:            m.id,
		characterId:   m.characterId,
		inventoryType: m.inventoryType,
		capacity:      m.capacity,
		assets:        m.assets,
	}
}

// SetCapacity sets the capacity field
func (b *builder) SetCapacity(capacity uint32) *builder {
	b.capacity = capacity
	return b
}

// AddAsset appends an asset to the assets slice
func (b *builder) AddAsset(a asset.Model) *builder {
	b.assets = append(b.assets, a)
	return b
}

// SetAssets replaces the assets slice
func (b *builder) SetAssets(as []asset.Model) *builder {
	b.assets = as
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrMissingId
	}
	return Model{
		id:            b.id,
		characterId:   b.characterId,
		inventoryType: b.inventoryType,
		capacity:      b.capacity,
		assets:        b.assets,
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
