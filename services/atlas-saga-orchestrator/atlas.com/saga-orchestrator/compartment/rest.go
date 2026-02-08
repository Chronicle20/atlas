package compartment

import (
	"strconv"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

// AssetRestModel represents an asset from the character inventory service (flattened)
type AssetRestModel struct {
	Id             string     `json:"-"`
	Slot           int16      `json:"slot"`
	TemplateId     uint32     `json:"templateId"`
	Expiration     time.Time  `json:"expiration"`
	CreatedAt      time.Time  `json:"createdAt"`
	Quantity       uint32     `json:"quantity"`
	OwnerId        uint32     `json:"ownerId"`
	Flag           uint16     `json:"flag"`
	Rechargeable   uint64     `json:"rechargeable"`
	Strength       uint16     `json:"strength"`
	Dexterity      uint16     `json:"dexterity"`
	Intelligence   uint16     `json:"intelligence"`
	Luck           uint16     `json:"luck"`
	Hp             uint16     `json:"hp"`
	Mp             uint16     `json:"mp"`
	WeaponAttack   uint16     `json:"weaponAttack"`
	MagicAttack    uint16     `json:"magicAttack"`
	WeaponDefense  uint16     `json:"weaponDefense"`
	MagicDefense   uint16     `json:"magicDefense"`
	Accuracy       uint16     `json:"accuracy"`
	Avoidability   uint16     `json:"avoidability"`
	Hands          uint16     `json:"hands"`
	Speed          uint16     `json:"speed"`
	Jump           uint16     `json:"jump"`
	Slots          uint16     `json:"slots"`
	Locked         bool       `json:"locked"`
	Spikes         bool       `json:"spikes"`
	KarmaUsed      bool       `json:"karmaUsed"`
	Cold           bool       `json:"cold"`
	CanBeTraded    bool       `json:"canBeTraded"`
	LevelType      byte       `json:"levelType"`
	Level          byte       `json:"level"`
	Experience     uint32     `json:"experience"`
	HammersApplied uint32     `json:"hammersApplied"`
	EquippedSince  *time.Time `json:"equippedSince"`
	CashId         int64      `json:"cashId,string"`
	CommodityId    uint32     `json:"commodityId"`
	PurchaseBy     uint32     `json:"purchaseBy"`
	PetId          uint32     `json:"petId"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = strconv.Itoa(id)
	return nil
}

// CompartmentRestModel represents a compartment from the character inventory service
type CompartmentRestModel struct {
	Id            string           `json:"-"`
	InventoryType byte             `json:"type"`
	Capacity      uint32           `json:"capacity"`
	Assets        []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id
}

func (r *CompartmentRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r CompartmentRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "assets",
			Name: "assets",
		},
	}
}

func (r CompartmentRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Assets {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

func (r CompartmentRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

func (r *CompartmentRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: strconv.Itoa(id)})
		}
	}
	return nil
}

func (r *CompartmentRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
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
