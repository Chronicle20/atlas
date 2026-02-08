package inventory

import (
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
	Id            uint32    `json:"-"`
	Slot          int16     `json:"slot"`
	TemplateId    uint32    `json:"templateId"`
	Expiration    time.Time `json:"expiration"`
	OwnerId       uint32    `json:"ownerId"`
	Strength      uint16    `json:"strength"`
	Dexterity     uint16    `json:"dexterity"`
	Intelligence  uint16    `json:"intelligence"`
	Luck          uint16    `json:"luck"`
	Hp            uint16    `json:"hp"`
	Mp            uint16    `json:"mp"`
	WeaponAttack  uint16    `json:"weaponAttack"`
	MagicAttack   uint16    `json:"magicAttack"`
	WeaponDefense uint16    `json:"weaponDefense"`
	MagicDefense  uint16    `json:"magicDefense"`
	Accuracy      uint16    `json:"accuracy"`
	Avoidability  uint16    `json:"avoidability"`
	Hands         uint16    `json:"hands"`
	Speed         uint16    `json:"speed"`
	Jump          uint16    `json:"jump"`
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

// IsEquipped returns true if this asset is in an equipped slot (negative slot number)
func (r AssetRestModel) IsEquipped() bool {
	return r.Slot < 0
}

// GetEquipableData returns the equipment stats from this asset's flat fields
func (r AssetRestModel) GetEquipableData() (EquipableRestData, bool) {
	if !r.IsEquipped() {
		return EquipableRestData{}, false
	}
	return EquipableRestData{
		OwnerId:       r.OwnerId,
		Strength:      r.Strength,
		Dexterity:     r.Dexterity,
		Intelligence:  r.Intelligence,
		Luck:          r.Luck,
		Hp:            r.Hp,
		Mp:            r.Mp,
		WeaponAttack:  r.WeaponAttack,
		MagicAttack:   r.MagicAttack,
		WeaponDefense: r.WeaponDefense,
		MagicDefense:  r.MagicDefense,
		Accuracy:      r.Accuracy,
		Avoidability:  r.Avoidability,
		Hands:         r.Hands,
		Speed:         r.Speed,
		Jump:          r.Jump,
	}, true
}
