package storage

import (
	"strconv"
	"time"
)

// AssetRestModel represents an asset from the storage service (flattened)
type AssetRestModel struct {
	Id             string     `json:"-"`
	Slot           int16      `json:"slot"`
	TemplateId     uint32     `json:"templateId"`
	Expiration     time.Time  `json:"expiration"`
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
	return "storage_assets"
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

// ProjectionAssetRestModel represents an asset from a storage projection (flattened)
type ProjectionAssetRestModel struct {
	Id             uint32     `json:"id"`
	Slot           int16      `json:"slot"`
	TemplateId     uint32     `json:"templateId"`
	Expiration     time.Time  `json:"expiration"`
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

func (r ProjectionAssetRestModel) GetName() string {
	return "storage_assets"
}

func (r ProjectionAssetRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *ProjectionAssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
