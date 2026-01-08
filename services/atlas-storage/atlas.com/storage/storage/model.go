package storage

import (
	"atlas-storage/asset"
	"errors"
	"github.com/google/uuid"
)

type Model struct {
	id        uuid.UUID
	worldId   byte
	accountId uint32
	capacity  uint32
	mesos     uint32
	assets    []asset.Model[any]
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) Capacity() uint32 {
	return m.capacity
}

func (m Model) Mesos() uint32 {
	return m.mesos
}

func (m Model) Assets() []asset.Model[any] {
	return m.assets
}

func (m Model) NextFreeSlot() (int16, error) {
	if uint32(len(m.assets)) >= m.capacity {
		return -1, errors.New("storage is full")
	}

	// Create a map of occupied slots
	occupied := make(map[int16]bool)
	for _, a := range m.assets {
		occupied[a.Slot()] = true
	}

	// Find the first free slot (1-indexed)
	for i := int16(1); i <= int16(m.capacity); i++ {
		if !occupied[i] {
			return i, nil
		}
	}

	return -1, errors.New("no free slot found")
}

func (m Model) HasCapacity() bool {
	return uint32(len(m.assets)) < m.capacity
}

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

func (b *ModelBuilder) Build() Model {
	return Model{
		id:        b.id,
		worldId:   b.worldId,
		accountId: b.accountId,
		capacity:  b.capacity,
		mesos:     b.mesos,
		assets:    b.assets,
	}
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
