package storage

import (
	"atlas-storage/asset"
	"errors"
	"github.com/google/uuid"
)

// ModelBuilder for constructing Model instances
type ModelBuilder struct {
	id        uuid.UUID
	worldId   byte
	accountId uint32
	capacity  uint32
	mesos     uint32
	assets    []asset.Model[any]
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		capacity: 4, // Default capacity
		assets:   make([]asset.Model[any], 0),
	}
}

func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetWorldId(worldId byte) *ModelBuilder {
	b.worldId = worldId
	return b
}

func (b *ModelBuilder) SetAccountId(accountId uint32) *ModelBuilder {
	b.accountId = accountId
	return b
}

func (b *ModelBuilder) SetCapacity(capacity uint32) *ModelBuilder {
	b.capacity = capacity
	return b
}

func (b *ModelBuilder) SetMesos(mesos uint32) *ModelBuilder {
	b.mesos = mesos
	return b
}

func (b *ModelBuilder) SetAssets(assets []asset.Model[any]) *ModelBuilder {
	b.assets = assets
	return b
}

func (b *ModelBuilder) validate() error {
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

func (b *ModelBuilder) Build() (Model, error) {
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
func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// Clone creates a copy of the Model with modifications
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:        m.id,
		worldId:   m.worldId,
		accountId: m.accountId,
		capacity:  m.capacity,
		mesos:     m.mesos,
		assets:    m.assets,
	}
}
