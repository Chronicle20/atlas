package storage

import (
	"atlas-storage/asset"
	"errors"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	id        uuid.UUID
	worldId   world.Id
	accountId uint32
	capacity  uint32
	mesos     uint32
	assets    []asset.Model[any]
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) WorldId() world.Id {
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

	// Find the first free slot (0-indexed)
	for i := int16(0); i < int16(m.capacity); i++ {
		if !occupied[i] {
			return i, nil
		}
	}

	return -1, errors.New("no free slot found")
}

func (m Model) HasCapacity() bool {
	return uint32(len(m.assets)) < m.capacity
}
