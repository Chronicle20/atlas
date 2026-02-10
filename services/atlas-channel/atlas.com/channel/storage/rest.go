package storage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
)

// StorageRestModel represents the storage REST response from atlas-storage
type StorageRestModel struct {
	Id        string           `json:"-"`
	WorldId   world.Id         `json:"world_id"`
	AccountId uint32           `json:"account_id"`
	Capacity  uint32           `json:"capacity"`
	Mesos     uint32           `json:"mesos"`
	Assets    []AssetRestModel `json:"-"`
}

func (r StorageRestModel) GetName() string {
	return "storages"
}

func (r StorageRestModel) GetID() string {
	return r.Id
}

func (r *StorageRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r StorageRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "storage_assets",
			Name: "assets",
		},
	}
}

func (r StorageRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Assets {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: "assets",
		})
	}
	return result
}

func (r *StorageRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *StorageRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *StorageRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["storage_assets"]; ok {
		assets := make([]AssetRestModel, 0)
		for _, ri := range r.Assets {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				assets = append(assets, wip)
			}
		}
		r.Assets = assets
	}
	return nil
}

// AssetRestModel represents a flat asset REST response from atlas-storage
type AssetRestModel struct {
	Id             uint32    `json:"id"`
	Slot           int16     `json:"slot"`
	TemplateId     uint32    `json:"templateId"`
	Expiration     time.Time `json:"expiration"`
	Quantity       uint32    `json:"quantity"`
	OwnerId        uint32    `json:"ownerId"`
	Flag           uint16    `json:"flag"`
	Rechargeable   uint64    `json:"rechargeable"`
	Strength       uint16    `json:"strength"`
	Dexterity      uint16    `json:"dexterity"`
	Intelligence   uint16    `json:"intelligence"`
	Luck           uint16    `json:"luck"`
	Hp             uint16    `json:"hp"`
	Mp             uint16    `json:"mp"`
	WeaponAttack   uint16    `json:"weaponAttack"`
	MagicAttack    uint16    `json:"magicAttack"`
	WeaponDefense  uint16    `json:"weaponDefense"`
	MagicDefense   uint16    `json:"magicDefense"`
	Accuracy       uint16    `json:"accuracy"`
	Avoidability   uint16    `json:"avoidability"`
	Hands          uint16    `json:"hands"`
	Speed          uint16    `json:"speed"`
	Jump           uint16    `json:"jump"`
	Slots     uint16 `json:"slots"`
	LevelType byte   `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	CashId         int64     `json:"cashId,string"`
	CommodityId    uint32    `json:"commodityId"`
	PurchaseBy     uint32    `json:"purchaseBy"`
	PetId          uint32    `json:"petId"`
}

func (r AssetRestModel) GetName() string {
	return "storage_assets"
}

func (r AssetRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *AssetRestModel) SetID(id string) error {
	intId, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(intId)
	return nil
}

// ProjectionRestModel represents a storage projection REST response from atlas-storage
type ProjectionRestModel struct {
	Id           string                     `json:"-"`
	CharacterId  uint32                     `json:"characterId"`
	AccountId    uint32                     `json:"accountId"`
	WorldId      world.Id                   `json:"worldId"`
	StorageId    string                     `json:"storageId"`
	Capacity     uint32                     `json:"capacity"`
	Mesos        uint32                     `json:"mesos"`
	NpcId        uint32                     `json:"npcId"`
	Compartments map[string]json.RawMessage `json:"compartments"`
}

func (r ProjectionRestModel) GetName() string {
	return "storage_projections"
}

func (r ProjectionRestModel) GetID() string {
	return r.Id
}

func (r *ProjectionRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ParseCompartmentAssets parses the raw compartment assets into typed AssetRestModel slices
func (r ProjectionRestModel) ParseCompartmentAssets() (map[string][]AssetRestModel, error) {
	result := make(map[string][]AssetRestModel)
	for name, rawAssets := range r.Compartments {
		var assets []AssetRestModel
		if err := json.Unmarshal(rawAssets, &assets); err != nil {
			return nil, fmt.Errorf("error unmarshaling compartment %s: %w", name, err)
		}
		result[name] = assets
	}
	return result, nil
}
