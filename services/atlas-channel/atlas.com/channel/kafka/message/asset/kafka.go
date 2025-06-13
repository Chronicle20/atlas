package asset

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	EnvEventTopicStatus            = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeUpdated         = "UPDATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeMoved           = "MOVED"
	StatusEventTypeQuantityChanged = "QUANTITY_CHANGED"
)

type StatusEvent[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	TemplateId    uint32    `json:"templateId"`
	Slot          int16     `json:"slot"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody[E any] struct {
	ReferenceId   uint32    `json:"referenceId"`
	ReferenceType string    `json:"referenceType"`
	ReferenceData E         `json:"referenceData"`
	Expiration    time.Time `json:"expiration"`
}

func convertMapToStruct[T any](in any) (T, error) {
	var out T
	b, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(b, &out)
	return out, err
}

func (b *CreatedStatusEventBody[E]) UnmarshalJSON(data []byte) error {
	type rawBody struct {
		ReferenceId   uint32          `json:"referenceId"`
		ReferenceType string          `json:"referenceType"`
		ReferenceData json.RawMessage `json:"referenceData"`
		Expiration    time.Time       `json:"expiration"`
	}

	var raw rawBody
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var typed any
	var err error
	if raw.ReferenceType == "equipable" {
		typed, err = convertMapToStruct[EquipableReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "cashEquipable" {
		typed, err = convertMapToStruct[CashEquipableReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "consumable" {
		typed, err = convertMapToStruct[ConsumableReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "setup" {
		typed, err = convertMapToStruct[SetupReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "etc" {
		typed, err = convertMapToStruct[EtcReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "cash" {
		typed, err = convertMapToStruct[CashReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "pet" {
		typed, err = convertMapToStruct[PetReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	var ref E
	ref, ok := typed.(E)
	if !ok {
		return fmt.Errorf("type assertion to E failed")
	}

	b.ReferenceId = raw.ReferenceId
	b.ReferenceType = raw.ReferenceType
	b.ReferenceData = ref
	b.Expiration = raw.Expiration

	return nil
}

type UpdatedStatusEventBody[E any] struct {
	ReferenceId   uint32    `json:"referenceId"`
	ReferenceType string    `json:"referenceType"`
	ReferenceData E         `json:"referenceData"`
	Expiration    time.Time `json:"expiration"`
}

func (b *UpdatedStatusEventBody[E]) UnmarshalJSON(data []byte) error {
	type rawBody struct {
		ReferenceId   uint32          `json:"referenceId"`
		ReferenceType string          `json:"referenceType"`
		ReferenceData json.RawMessage `json:"referenceData"`
		Expiration    time.Time       `json:"expiration"`
	}

	var raw rawBody
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var typed any
	var err error
	if raw.ReferenceType == "equipable" {
		typed, err = convertMapToStruct[EquipableReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "cashEquipable" {
		typed, err = convertMapToStruct[CashEquipableReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "consumable" {
		typed, err = convertMapToStruct[ConsumableReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "setup" {
		typed, err = convertMapToStruct[SetupReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "etc" {
		typed, err = convertMapToStruct[EtcReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "cash" {
		typed, err = convertMapToStruct[CashReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	if raw.ReferenceType == "pet" {
		typed, err = convertMapToStruct[PetReferenceData](raw.ReferenceData)
		if err != nil {
			return err
		}
	}
	var ref E
	ref, ok := typed.(E)
	if !ok {
		return fmt.Errorf("type assertion to E failed")
	}

	b.ReferenceId = raw.ReferenceId
	b.ReferenceType = raw.ReferenceType
	b.ReferenceData = ref
	b.Expiration = raw.Expiration

	return nil
}

type EquipableReferenceData struct {
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
	Slots          uint16    `json:"slots"`
	OwnerId        uint32    `json:"ownerId"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}

type CashEquipableReferenceData struct {
	CashId         int64     `json:"cashId,string"`
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
	Slots          uint16    `json:"slots"`
	OwnerId        uint32    `json:"ownerId"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}

type ConsumableReferenceData struct {
	Quantity     uint32 `json:"quantity"`
	OwnerId      uint32 `json:"ownerId"`
	Flag         uint16 `json:"flag"`
	Rechargeable uint64 `json:"rechargeable"`
}

type SetupReferenceData struct {
	Quantity uint32 `json:"quantity"`
	OwnerId  uint32 `json:"ownerId"`
	Flag     uint16 `json:"flag"`
}

type EtcReferenceData struct {
	Quantity uint32 `json:"quantity"`
	OwnerId  uint32 `json:"ownerId"`
	Flag     uint16 `json:"flag"`
}

type CashReferenceData struct {
	CashId      int64  `json:"cashId,string"`
	Quantity    uint32 `json:"quantity"`
	OwnerId     uint32 `json:"ownerId"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
}

type PetReferenceData struct {
	CashId      uint64 `json:"cashId"`
	OwnerId     uint32 `json:"ownerId"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
	Name        string `json:"name"`
	Level       byte   `json:"level"`
	Closeness   uint16 `json:"closeness"`
	Fullness    byte   `json:"fullness"`
	Slot        int8   `json:"slot"`
}

type DeletedStatusEventBody struct {
}

type MovedStatusEventBody struct {
	OldSlot int16 `json:"oldSlot"`
}

type QuantityChangedEventBody struct {
	Quantity uint32 `json:"quantity"`
}
