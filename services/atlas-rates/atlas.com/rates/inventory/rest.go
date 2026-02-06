package inventory

import (
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

// Compartment type constants matching atlas-constants/inventory
const (
	CompartmentTypeEquip uint8 = 1
	CompartmentTypeUse   uint8 = 2
	CompartmentTypeSetup uint8 = 3
	CompartmentTypeETC   uint8 = 4
	CompartmentTypeCash  uint8 = 5
)

// RestModel represents an inventory from atlas-inventory
type RestModel struct {
	Id           string                 `json:"-"`
	CharacterId  uint32                 `json:"characterId"`
	Compartments []CompartmentRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "inventories"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "compartments",
			Name: "compartments",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Compartments {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Compartments {
		result = append(result, r.Compartments[key])
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "compartments" {
		for _, idStr := range IDs {
			r.Compartments = append(r.Compartments, CompartmentRestModel{Id: idStr})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["compartments"]; ok {
		compartments := make([]CompartmentRestModel, 0)
		for _, ri := range r.Compartments {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				compartments = append(compartments, wip)
			}
		}
		r.Compartments = compartments
	}
	return nil
}

// CompartmentRestModel represents a compartment within an inventory
type CompartmentRestModel struct {
	Id       string           `json:"-"`
	Type     uint8            `json:"type"`
	Capacity uint32           `json:"capacity"`
	Assets   []AssetRestModel `json:"-"`
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
			r.Assets = append(r.Assets, AssetRestModel{Id: idStr})
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

// AssetRestModel represents an asset from atlas-inventory
type AssetRestModel struct {
	Id            string                 `json:"-"`
	TemplateId    uint32                 `json:"templateId"`
	Slot          int16                  `json:"slot"`
	ReferenceId   uint32                 `json:"referenceId"`
	ReferenceType string                 `json:"referenceType"`
	ReferenceData map[string]interface{} `json:"referenceData"`
	Expiration    time.Time              `json:"expiration"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *AssetRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *AssetRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// GetCreatedAt extracts the createdAt timestamp from ReferenceData if present
func (r AssetRestModel) GetCreatedAt() time.Time {
	if r.ReferenceData == nil {
		return time.Time{}
	}
	if createdAtStr, ok := r.ReferenceData["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			return t
		}
	}
	return time.Time{}
}

// IsEquipmentSlot returns true if the slot is an equipment slot (negative)
func (r AssetRestModel) IsEquipmentSlot() bool {
	return r.Slot < 0
}

// IsEquipable returns true if the asset is an equipable item
func (r AssetRestModel) IsEquipable() bool {
	return r.ReferenceType == "EQUIPABLE" || r.ReferenceType == "equipable"
}

// IsCash returns true if the asset is a cash item
func (r AssetRestModel) IsCash() bool {
	return r.ReferenceType == "CASH" || r.ReferenceType == "cash"
}

// GetEquippedSince extracts the equippedSince timestamp from ReferenceData if present
// Returns nil if not equipped or not an equipable
func (r AssetRestModel) GetEquippedSince() *time.Time {
	if r.ReferenceData == nil {
		return nil
	}
	if equippedSinceStr, ok := r.ReferenceData["equippedSince"].(string); ok {
		if t, err := time.Parse(time.RFC3339, equippedSinceStr); err == nil {
			return &t
		}
	}
	return nil
}
