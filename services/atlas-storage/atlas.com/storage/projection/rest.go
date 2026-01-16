package projection

import (
	"atlas-storage/asset"
	"strconv"
)

// RestModel represents a projection in JSON:API format
type RestModel struct {
	Id           string                              `json:"-"`
	CharacterId  uint32                              `json:"characterId"`
	AccountId    uint32                              `json:"accountId"`
	WorldId      byte                                `json:"worldId"`
	StorageId    string                              `json:"storageId"`
	Capacity     uint32                              `json:"capacity"`
	Mesos        uint32                              `json:"mesos"`
	NpcId        uint32                              `json:"npcId"`
	Compartments map[string][]asset.BaseRestModel   `json:"compartments"`
}

func (r RestModel) GetName() string {
	return "storage_projections"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	compartments := make(map[string][]asset.BaseRestModel)

	for invType, assets := range m.Compartments() {
		restAssets, err := asset.TransformAllToBaseRestModel(assets)
		if err != nil {
			return RestModel{}, err
		}
		// Use inventory type name as key
		compartments[inventoryTypeName(invType)] = restAssets
	}

	return RestModel{
		Id:           strconv.Itoa(int(m.CharacterId())),
		CharacterId:  m.CharacterId(),
		AccountId:    m.AccountId(),
		WorldId:      m.WorldId(),
		StorageId:    m.StorageId().String(),
		Capacity:     m.Capacity(),
		Mesos:        m.Mesos(),
		NpcId:        m.NpcId(),
		Compartments: compartments,
	}, nil
}

func inventoryTypeName(t asset.InventoryType) string {
	switch t {
	case asset.InventoryTypeEquip:
		return "equip"
	case asset.InventoryTypeUse:
		return "use"
	case asset.InventoryTypeSetup:
		return "setup"
	case asset.InventoryTypeEtc:
		return "etc"
	case asset.InventoryTypeCash:
		return "cash"
	default:
		return "unknown"
	}
}

func inventoryTypeFromName(name string) asset.InventoryType {
	switch name {
	case "equip":
		return asset.InventoryTypeEquip
	case "use":
		return asset.InventoryTypeUse
	case "setup":
		return asset.InventoryTypeSetup
	case "etc":
		return asset.InventoryTypeEtc
	case "cash":
		return asset.InventoryTypeCash
	default:
		return 0
	}
}

func inventoryTypeFromByte(b byte) asset.InventoryType {
	return asset.InventoryType(b)
}
