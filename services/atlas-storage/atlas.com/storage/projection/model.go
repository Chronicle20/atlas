package projection

import (
	"atlas-storage/asset"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

func (m Model) MarshalJSON() ([]byte, error) {
	// Convert map[inventory.Type][]asset.Model to map[string][]asset.Model for JSON
	compartments := make(map[string][]asset.Model, len(m.compartments))
	for k, v := range m.compartments {
		compartments[strconv.Itoa(int(k))] = v
	}
	return json.Marshal(struct {
		CharacterId  uint32                   `json:"characterId"`
		AccountId    uint32                   `json:"accountId"`
		WorldId      world.Id                 `json:"worldId"`
		StorageId    uuid.UUID                `json:"storageId"`
		Capacity     uint32                   `json:"capacity"`
		Mesos        uint32                   `json:"mesos"`
		NpcId        uint32                   `json:"npcId"`
		Compartments map[string][]asset.Model `json:"compartments"`
	}{
		CharacterId:  m.characterId,
		AccountId:    m.accountId,
		WorldId:      m.worldId,
		StorageId:    m.storageId,
		Capacity:     m.capacity,
		Mesos:        m.mesos,
		NpcId:        m.npcId,
		Compartments: compartments,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		CharacterId  uint32                   `json:"characterId"`
		AccountId    uint32                   `json:"accountId"`
		WorldId      world.Id                 `json:"worldId"`
		StorageId    uuid.UUID                `json:"storageId"`
		Capacity     uint32                   `json:"capacity"`
		Mesos        uint32                   `json:"mesos"`
		NpcId        uint32                   `json:"npcId"`
		Compartments map[string][]asset.Model `json:"compartments"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.characterId = aux.CharacterId
	m.accountId = aux.AccountId
	m.worldId = aux.WorldId
	m.storageId = aux.StorageId
	m.capacity = aux.Capacity
	m.mesos = aux.Mesos
	m.npcId = aux.NpcId
	m.compartments = make(map[inventory.Type][]asset.Model, len(aux.Compartments))
	for k, v := range aux.Compartments {
		typeInt, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		m.compartments[inventory.Type(typeInt)] = v
	}
	return nil
}
