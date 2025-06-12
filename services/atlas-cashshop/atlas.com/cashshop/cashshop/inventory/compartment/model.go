package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"github.com/google/uuid"
)

// CompartmentType represents the type of cash shop inventory compartment
type CompartmentType string

const (
	TypeExplorer = CompartmentType("explorer")
	TypeCygnus   = CompartmentType("cygnus")
	TypeLegend   = CompartmentType("legend")
)

// Model represents a cash shop inventory compartment
type Model struct {
	id         uuid.UUID
	accountId  uint32
	type_      CompartmentType
	capacity   uint32
	assets     []asset.Model
}

// Id returns the unique identifier of this compartment
func (m Model) Id() uuid.UUID {
	return m.id
}

// AccountId returns the account ID associated with this compartment
func (m Model) AccountId() uint32 {
	return m.accountId
}

// Type returns the type of this compartment
func (m Model) Type() CompartmentType {
	return m.type_
}

// Capacity returns the maximum number of items this compartment can hold
func (m Model) Capacity() uint32 {
	return m.capacity
}

// Assets returns all assets in this compartment
func (m Model) Assets() []asset.Model {
	return m.assets
}

// FindById finds an asset by its ID
func (m Model) FindById(id uuid.UUID) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if a.Id() == id {
			return &a, true
		}
	}
	return nil, false
}

// FindByTemplateId finds an asset by its template ID
func (m Model) FindByTemplateId(templateId uint32) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if a.TemplateId() == templateId {
			return &a, true
		}
	}
	return nil, false
}

// Clone creates a builder from this model
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:         m.id,
		accountId:  m.accountId,
		type_:      m.type_,
		capacity:   m.capacity,
		assets:     m.assets,
	}
}

// ModelBuilder is a builder for the Model
type ModelBuilder struct {
	id         uuid.UUID
	accountId  uint32
	type_      CompartmentType
	capacity   uint32
	assets     []asset.Model
}

// NewBuilder creates a new ModelBuilder
func NewBuilder(id uuid.UUID, accountId uint32, type_ CompartmentType, capacity uint32) *ModelBuilder {
	return &ModelBuilder{
		id:         id,
		accountId:  accountId,
		type_:      type_,
		capacity:   capacity,
		assets:     make([]asset.Model, 0),
	}
}

// SetCapacity sets the capacity of this compartment
func (b *ModelBuilder) SetCapacity(capacity uint32) *ModelBuilder {
	b.capacity = capacity
	return b
}

// AddAsset adds an asset to this compartment
func (b *ModelBuilder) AddAsset(a asset.Model) *ModelBuilder {
	b.assets = append(b.assets, a)
	return b
}

// SetAssets sets all assets in this compartment
func (b *ModelBuilder) SetAssets(as []asset.Model) *ModelBuilder {
	b.assets = as
	return b
}

// Build creates a Model from this builder
func (b *ModelBuilder) Build() Model {
	return Model{
		id:         b.id,
		accountId:  b.accountId,
		type_:      b.type_,
		capacity:   b.capacity,
		assets:     b.assets,
	}
}
