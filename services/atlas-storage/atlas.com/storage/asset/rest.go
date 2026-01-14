package asset

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type BaseRestModel struct {
	Id            uint32      `json:"-"`
	Slot          int16       `json:"slot"`
	TemplateId    uint32      `json:"templateId"`
	Expiration    time.Time   `json:"expiration"`
	ReferenceId   uint32      `json:"referenceId"`
	ReferenceType string      `json:"referenceType"`
	ReferenceData interface{} `json:"referenceData"`
}

func (r BaseRestModel) GetName() string {
	return "storage_assets"
}

func (r BaseRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *BaseRestModel) SetID(id string) error {
	intId, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(intId)
	return nil
}

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

type CashEquipableRestData struct {
	CashBaseRestData
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

func (r *BaseRestModel) UnmarshalJSON(data []byte) error {
	type Alias BaseRestModel
	temp := &struct {
		*Alias
		ReferenceData json.RawMessage `json:"referenceData"`
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if ReferenceType(temp.ReferenceType) == ReferenceTypeEquipable {
		var rd EquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeEquipable, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeCashEquipable {
		var rd CashEquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeCashEquipable, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeConsumable {
		var rd ConsumableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeConsumable, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeSetup {
		var rd SetupRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeSetup, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeEtc {
		var rd EtcRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeEtc, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeCash {
		var rd CashRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeCash, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypePet {
		var rd PetRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypePet, err)
		}
		r.ReferenceData = rd
	}
	return nil
}

// RestModel represents the legacy format (without reference data)
type RestModel struct {
	Id            string    `json:"-"`
	StorageId     string    `json:"storage_id"`
	InventoryType byte      `json:"inventory_type"`
	Slot          int16     `json:"slot"`
	TemplateId    uint32    `json:"template_id"`
	Expiration    time.Time `json:"expiration"`
	ReferenceId   uint32    `json:"reference_id"`
	ReferenceType string    `json:"reference_type"`
	// Stackable data (only present for stackable items)
	Quantity *uint32 `json:"quantity,omitempty"`
	OwnerId  *uint32 `json:"owner_id,omitempty"`
	Flag     *uint16 `json:"flag,omitempty"`
}

func (r RestModel) GetName() string {
	return "storage_assets"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Transform converts a Model to a RestModel (legacy format without reference data)
func Transform(m Model[any]) RestModel {
	return RestModel{
		Id:            strconv.Itoa(int(m.Id())),
		StorageId:     m.StorageId().String(),
		InventoryType: byte(m.InventoryType()),
		Slot:          m.Slot(),
		TemplateId:    m.TemplateId(),
		Expiration:    m.Expiration(),
		ReferenceId:   m.ReferenceId(),
		ReferenceType: string(m.ReferenceType()),
	}
}

// TransformWithStackable converts a Model to a RestModel with stackable data (legacy format)
func TransformWithStackable(m Model[any], quantity uint32, ownerId uint32, flag uint16) RestModel {
	rm := Transform(m)
	rm.Quantity = &quantity
	rm.OwnerId = &ownerId
	rm.Flag = &flag
	return rm
}

// TransformAll converts multiple Models to RestModels (legacy format)
func TransformAll(models []Model[any]) []RestModel {
	result := make([]RestModel, 0, len(models))
	for _, m := range models {
		result = append(result, Transform(m))
	}
	return result
}
