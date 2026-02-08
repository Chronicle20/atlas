package compartment

import (
	"atlas-channel/asset"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

var (
	ErrMissingId = errors.New("compartment id is required")
)

type modelBuilder struct {
	id            uuid.UUID
	characterId   uint32
	inventoryType inventory.Type
	capacity      uint32
	assets        []asset.Model
}

// NewModelBuilder creates a new builder instance with required fields
func NewModelBuilder(id uuid.UUID, characterId uint32, it inventory.Type, capacity uint32) *modelBuilder {
	return &modelBuilder{
		id:            id,
		characterId:   characterId,
		inventoryType: it,
		capacity:      capacity,
		assets:        make([]asset.Model, 0),
	}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder(id uuid.UUID, characterId uint32, it inventory.Type, capacity uint32) *modelBuilder {
	return NewModelBuilder(id, characterId, it, capacity)
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:            m.id,
		characterId:   m.characterId,
		inventoryType: m.inventoryType,
		capacity:      m.capacity,
		assets:        m.assets,
	}
}

// SetCapacity sets the capacity field
func (b *modelBuilder) SetCapacity(capacity uint32) *modelBuilder {
	b.capacity = capacity
	return b
}

// AddAsset appends an asset to the assets slice
func (b *modelBuilder) AddAsset(a asset.Model) *modelBuilder {
	b.assets = append(b.assets, a)
	return b
}

// SetAssets replaces the assets slice
func (b *modelBuilder) SetAssets(as []asset.Model) *modelBuilder {
	b.assets = as
	return b
}

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
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
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
