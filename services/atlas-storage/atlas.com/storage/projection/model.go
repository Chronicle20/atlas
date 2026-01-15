package projection

import (
	"atlas-storage/asset"
	"errors"
	"github.com/google/uuid"
)

// Model represents an in-memory projection of storage state for a character session.
// Each compartment slice initially contains ALL assets; filtering occurs on operations.
type Model struct {
	characterId  uint32
	accountId    uint32
	worldId      byte
	storageId    uuid.UUID
	capacity     uint32
	mesos        uint32
	npcId        uint32
	compartments map[asset.InventoryType][]asset.Model[any]
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) WorldId() byte {
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

func (m Model) Compartments() map[asset.InventoryType][]asset.Model[any] {
	return m.compartments
}

// GetCompartment returns the asset slice for a specific inventory type
func (m Model) GetCompartment(inventoryType asset.InventoryType) []asset.Model[any] {
	if assets, ok := m.compartments[inventoryType]; ok {
		return assets
	}
	return []asset.Model[any]{}
}

// GetAssetBySlot returns the asset at the given slot (index) in the compartment
func (m Model) GetAssetBySlot(inventoryType asset.InventoryType, slot int16) (asset.Model[any], bool) {
	assets := m.GetCompartment(inventoryType)
	if slot < 0 || int(slot) >= len(assets) {
		return asset.Model[any]{}, false
	}
	return assets[slot], true
}

// AllCompartmentTypes returns all valid inventory types
func AllCompartmentTypes() []asset.InventoryType {
	return []asset.InventoryType{
		asset.InventoryTypeEquip,
		asset.InventoryTypeUse,
		asset.InventoryTypeSetup,
		asset.InventoryTypeEtc,
		asset.InventoryTypeCash,
	}
}

// Builder for constructing Model instances
type Builder struct {
	characterId  uint32
	accountId    uint32
	worldId      byte
	storageId    uuid.UUID
	capacity     uint32
	mesos        uint32
	npcId        uint32
	compartments map[asset.InventoryType][]asset.Model[any]
}

func NewBuilder() *Builder {
	return &Builder{
		compartments: make(map[asset.InventoryType][]asset.Model[any]),
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

func (b *Builder) SetWorldId(worldId byte) *Builder {
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

func (b *Builder) SetCompartments(compartments map[asset.InventoryType][]asset.Model[any]) *Builder {
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
	compartments := make(map[asset.InventoryType][]asset.Model[any])
	for k, v := range m.compartments {
		copied := make([]asset.Model[any], len(v))
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

// RemoveAssetFromCompartment removes an asset at the given slot and filters the compartment
// to only include assets matching the compartment's inventory type.
// Returns the updated compartment slice.
func RemoveAssetFromCompartment(compartment []asset.Model[any], slot int16, inventoryType asset.InventoryType) []asset.Model[any] {
	if slot < 0 || int(slot) >= len(compartment) {
		return compartment
	}

	// Remove the asset at slot
	result := make([]asset.Model[any], 0, len(compartment)-1)
	for i, a := range compartment {
		if int16(i) == slot {
			continue
		}
		// Filter to only matching inventory types
		if a.InventoryType() == inventoryType {
			result = append(result, a)
		}
	}
	return result
}

// AddAssetToCompartment adds an asset and filters the compartment
// to only include assets matching the compartment's inventory type.
// Returns the updated compartment slice with proper slot ordering.
func AddAssetToCompartment(compartment []asset.Model[any], newAsset asset.Model[any], inventoryType asset.InventoryType) []asset.Model[any] {
	// Filter existing assets to only matching inventory types and add the new asset
	result := make([]asset.Model[any], 0, len(compartment)+1)

	for _, a := range compartment {
		if a.InventoryType() == inventoryType {
			result = append(result, a)
		}
	}

	// Add the new asset if it matches the inventory type
	if newAsset.InventoryType() == inventoryType {
		result = append(result, newAsset)
	}

	return result
}
