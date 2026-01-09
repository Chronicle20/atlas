package storage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// StorageRestModel represents the storage REST response from atlas-storage
type StorageRestModel struct {
	Id        string             `json:"-"`
	WorldId   byte               `json:"world_id"`
	AccountId uint32             `json:"account_id"`
	Capacity  uint32             `json:"capacity"`
	Mesos     uint32             `json:"mesos"`
	Assets    []AssetRestModel   `json:"assets"`
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

// AssetRestModel represents an asset REST response from atlas-storage with full reference data
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

// Reference data types matching inventory service format
type BaseData struct {
	OwnerId uint32 `json:"ownerId"`
}

type StatisticRestData struct {
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

type CashBaseRestData struct {
	CashId int64 `json:"cashId,string"`
}

type StackableRestData struct {
	Quantity uint32 `json:"quantity"`
}

type EquipableRestData struct {
	BaseData
	StatisticRestData
	Slots          uint16 `json:"slots"`
	Locked         bool   `json:"locked"`
	Spikes         bool   `json:"spikes"`
	KarmaUsed      bool   `json:"karmaUsed"`
	Cold           bool   `json:"cold"`
	CanBeTraded    bool   `json:"canBeTraded"`
	LevelType      byte   `json:"levelType"`
	Level          byte   `json:"level"`
	Experience     uint32 `json:"experience"`
	HammersApplied uint32 `json:"hammersApplied"`
}

type ConsumableRestData struct {
	BaseData
	StackableRestData
	Flag         uint16 `json:"flag"`
	Rechargeable uint64 `json:"rechargeable"`
}

type SetupRestData struct {
	BaseData
	StackableRestData
	Flag uint16 `json:"flag"`
}

type EtcRestData struct {
	BaseData
	StackableRestData
	Flag uint16 `json:"flag"`
}

type CashRestData struct {
	BaseData
	CashBaseRestData
	StackableRestData
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
}

type PetRestData struct {
	BaseData
	CashBaseRestData
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
	Name        string `json:"name"`
	Level       byte   `json:"level"`
	Closeness   uint16 `json:"closeness"`
	Fullness    byte   `json:"fullness"`
	Slot        int8   `json:"slot"`
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

	// Unmarshal reference data based on type
	switch temp.ReferenceType {
	case "equipable":
		var rd EquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling equipable referenceData: %w", err)
		}
		r.ReferenceData = rd
	case "cash_equipable":
		var rd EquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling cash_equipable referenceData: %w", err)
		}
		r.ReferenceData = rd
	case "consumable":
		var rd ConsumableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling consumable referenceData: %w", err)
		}
		r.ReferenceData = rd
	case "setup":
		var rd SetupRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling setup referenceData: %w", err)
		}
		r.ReferenceData = rd
	case "etc":
		var rd EtcRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling etc referenceData: %w", err)
		}
		r.ReferenceData = rd
	case "cash":
		var rd CashRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling cash referenceData: %w", err)
		}
		r.ReferenceData = rd
	case "pet":
		var rd PetRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling pet referenceData: %w", err)
		}
		r.ReferenceData = rd
	}

	return nil
}
