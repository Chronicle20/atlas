package projection

import (
	"atlas-storage/asset"
	"strconv"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/world"
)

// RestModel represents a projection in JSON:API format
type RestModel struct {
	Id           string                           `json:"-"`
	CharacterId  uint32                           `json:"characterId"`
	AccountId    uint32                           `json:"accountId"`
	WorldId      world.Id                         `json:"worldId"`
	StorageId    string                           `json:"storageId"`
	Capacity     uint32                           `json:"capacity"`
	Mesos        uint32                           `json:"mesos"`
	NpcId        uint32                           `json:"npcId"`
	Compartments map[string][]asset.RestModel     `json:"compartments"`
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
	compartments := make(map[string][]asset.RestModel)

	for invType, assets := range m.Compartments() {
		restAssets, err := asset.TransformAll(assets)
		if err != nil {
			return RestModel{}, err
		}
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

func inventoryTypeName(t inventory.Type) string {
	switch t {
	case inventory.TypeValueEquip:
		return "equip"
	case inventory.TypeValueUse:
		return "use"
	case inventory.TypeValueSetup:
		return "setup"
	case inventory.TypeValueETC:
		return "etc"
	case inventory.TypeValueCash:
		return "cash"
	default:
		return "unknown"
	}
}

func inventoryTypeFromName(name string) inventory.Type {
	switch name {
	case "equip":
		return inventory.TypeValueEquip
	case "use":
		return inventory.TypeValueUse
	case "setup":
		return inventory.TypeValueSetup
	case "etc":
		return inventory.TypeValueETC
	case "cash":
		return inventory.TypeValueCash
	default:
		return 0
	}
}
