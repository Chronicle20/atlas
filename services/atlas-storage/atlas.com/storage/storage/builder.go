package storage

import (
	"atlas-storage/asset"
	"errors"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder for constructing Model instances
type Builder struct {
	id        uuid.UUID
	worldId   world.Id
	accountId uint32
	capacity  uint32
	mesos     uint32
	assets    []asset.Model
}

func NewBuilder() *Builder {
	return &Builder{
		capacity: 4, // Default capacity
		assets:   make([]asset.Model, 0),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetAccountId(accountId uint32) *Builder {
	b.accountId = accountId
	return b
}

func (b *Builder) SetCapacity(capacity uint32) *Builder {
	b.capacity = capacity
	return b
}

func (b *Builder) SetMesos(mesos uint32) *Builder {
	b.mesos = mesos
	return b
}

func (b *Builder) SetAssets(assets []asset.Model) *Builder {
	b.assets = assets
	return b
}

func (b *Builder) validate() error {
	if b.id == uuid.Nil {
		return errors.New("storage id is required")
	}
	if b.accountId == 0 {
		return errors.New("account id is required")
	}
	if b.capacity == 0 {
		return errors.New("capacity must be greater than 0")
	}
	return nil
}

func (b *Builder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		id:        b.id,
		worldId:   b.worldId,
		accountId: b.accountId,
		capacity:  b.capacity,
		mesos:     b.mesos,
		assets:    b.assets,
	}, nil
}

// MustBuild builds the model, panicking on validation error.
// Use only for trusted internal data (e.g., from database entities).
func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// Clone creates a copy of the Model with modifications
func Clone(m Model) *Builder {
	return &Builder{
		id:        m.id,
		worldId:   m.worldId,
		accountId: m.accountId,
		capacity:  m.capacity,
		mesos:     m.mesos,
		assets:    m.assets,
	}
}
