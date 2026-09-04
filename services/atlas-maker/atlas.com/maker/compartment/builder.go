package compartment

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// Builder constructs a Model directly, for tests that need a compartment
// snapshot with specific assets without standing up an httptest server for
// atlas-inventory's wire format.
type Builder struct {
	id            uuid.UUID
	inventoryType inventory.Type
	capacity      uint32
	assets        []AssetModel
}

func NewBuilder(inventoryType inventory.Type) *Builder {
	return &Builder{id: uuid.New(), inventoryType: inventoryType}
}

func (b *Builder) SetCapacity(capacity uint32) *Builder {
	b.capacity = capacity
	return b
}

func (b *Builder) AddAsset(asset AssetModel) *Builder {
	b.assets = append(b.assets, asset)
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:            b.id,
		inventoryType: b.inventoryType,
		capacity:      b.capacity,
		assets:        b.assets,
	}
}
