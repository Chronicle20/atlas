package inventory

import (
	"time"
)

// RestModel represents an inventory from atlas-inventory
type RestModel struct {
	Id           string                `json:"-"`
	CharacterId  uint32                `json:"characterId"`
	Compartments []CompartmentRestModel `json:"compartments"`
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

// CompartmentRestModel represents a compartment within an inventory
type CompartmentRestModel struct {
	Id     string           `json:"id"`
	Type   string           `json:"type"`
	Assets []AssetRestModel `json:"assets"`
}

// AssetRestModel represents an asset from atlas-inventory
type AssetRestModel struct {
	Id            string                 `json:"-"`
	CompartmentId string                 `json:"compartmentId"`
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
