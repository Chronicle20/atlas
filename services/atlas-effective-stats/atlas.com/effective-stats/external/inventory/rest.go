package inventory

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// CompartmentRestModel represents a compartment from atlas-inventory service
type CompartmentRestModel struct {
	Id            string           `json:"-"`
	InventoryType int8             `json:"type"`
	Capacity      uint32           `json:"capacity"`
	Assets        []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id
}

func (r *CompartmentRestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
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

// AssetRestModel represents an asset from atlas-inventory service
type AssetRestModel struct {
	Id            uint32      `json:"-"`
	Slot          int16       `json:"slot"`
	TemplateId    uint32      `json:"templateId"`
	Expiration    time.Time   `json:"expiration"`
	ReferenceId   uint32      `json:"referenceId"`
	ReferenceType string      `json:"referenceType"`
	ReferenceData interface{} `json:"referenceData"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

const (
	ReferenceTypeEquipable     = "equipable"
	ReferenceTypeCashEquipable = "cash_equipable"
)

// EquipableRestData contains equipment stats
type EquipableRestData struct {
	OwnerId       uint32 `json:"ownerId"`
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	Hp            uint16 `json:"hp"`
	Mp            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
}

func (r *AssetRestModel) UnmarshalJSON(data []byte) error {
	type Alias AssetRestModel
	temp := &struct {
		*Alias
		ReferenceData json.RawMessage `json:"referenceData"`
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if r.ReferenceType == ReferenceTypeEquipable || r.ReferenceType == ReferenceTypeCashEquipable {
		var rd EquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", r.ReferenceType, err)
		}
		r.ReferenceData = rd
	}
	return nil
}

// IsEquipped returns true if this asset is in an equipped slot (negative slot number)
func (r AssetRestModel) IsEquipped() bool {
	return r.Slot < 0
}

// GetEquipableData returns the equipment stats if this is an equipable item
func (r AssetRestModel) GetEquipableData() (EquipableRestData, bool) {
	if r.ReferenceType == ReferenceTypeEquipable || r.ReferenceType == ReferenceTypeCashEquipable {
		if data, ok := r.ReferenceData.(EquipableRestData); ok {
			return data, true
		}
	}
	return EquipableRestData{}, false
}
