package projection

import (
	"atlas-storage/asset"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// Model represents an in-memory projection of storage state for a character session.
type Model struct {
	characterId  uint32
	accountId    uint32
	worldId      world.Id
	storageId    uuid.UUID
	capacity     uint32
	mesos        uint32
	npcId        uint32
	compartments map[inventory.Type][]asset.Model
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) StorageId() uuid.UUID {
	return m.storageId
}

func (m Model) Capacity() uint32 {
	return m.capacity
}

func (m Model) Mesos() uint32 {
	return m.mesos
}

func (m Model) NpcId() uint32 {
	return m.npcId
}

func (m Model) Compartments() map[inventory.Type][]asset.Model {
	return m.compartments
}

// GetCompartment returns the asset slice for a specific inventory type
func (m Model) GetCompartment(inventoryType inventory.Type) []asset.Model {
	if assets, ok := m.compartments[inventoryType]; ok {
		return assets
	}
	return []asset.Model{}
}

// GetAssetBySlot returns the asset at the given slot (index) in the compartment
func (m Model) GetAssetBySlot(inventoryType inventory.Type, slot int16) (asset.Model, bool) {
	assets := m.GetCompartment(inventoryType)
	if slot < 0 || int(slot) >= len(assets) {
		return asset.Model{}, false
	}
	return assets[slot], true
}

// Builder for constructing Model instances
type Builder struct {
	characterId  uint32
	accountId    uint32
	worldId      world.Id
	storageId    uuid.UUID
	capacity     uint32
	mesos        uint32
	npcId        uint32
	compartments map[inventory.Type][]asset.Model
}

func NewBuilder() *Builder {
	return &Builder{
		compartments: make(map[inventory.Type][]asset.Model),
	}
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetAccountId(accountId uint32) *Builder {
	b.accountId = accountId
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetStorageId(storageId uuid.UUID) *Builder {
	b.storageId = storageId
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

func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

func (b *Builder) SetCompartments(compartments map[inventory.Type][]asset.Model) *Builder {
	b.compartments = compartments
	return b
}

func (b *Builder) validate() error {
	if b.characterId == 0 {
		return errors.New("character id is required")
	}
	if b.accountId == 0 {
		return errors.New("account id is required")
	}
	if b.storageId == uuid.Nil {
		return errors.New("storage id is required")
	}
	return nil
}

func (b *Builder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		characterId:  b.characterId,
		accountId:    b.accountId,
		worldId:      b.worldId,
		storageId:    b.storageId,
		capacity:     b.capacity,
		mesos:        b.mesos,
		npcId:        b.npcId,
		compartments: b.compartments,
	}, nil
}

func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// Clone creates a builder from an existing model for modifications
func Clone(m Model) *Builder {
	// Deep copy compartments
	compartments := make(map[inventory.Type][]asset.Model)
	for k, v := range m.compartments {
		copied := make([]asset.Model, len(v))
		copy(copied, v)
		compartments[k] = copied
	}

	return &Builder{
		characterId:  m.characterId,
		accountId:    m.accountId,
		worldId:      m.worldId,
		storageId:    m.storageId,
		capacity:     m.capacity,
		mesos:        m.mesos,
		npcId:        m.npcId,
		compartments: compartments,
	}
}
