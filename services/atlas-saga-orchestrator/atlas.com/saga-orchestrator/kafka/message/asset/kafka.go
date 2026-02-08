package asset

import (
	"time"

	"github.com/google/uuid"
)

const (
	EnvEventTopicStatus            = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeMoved           = "MOVED"
	StatusEventTypeQuantityChanged = "QUANTITY_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetId       uint32    `json:"assetId"`
	TemplateId    uint32    `json:"templateId"`
	Slot          int16     `json:"slot"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type AssetData struct {
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

type CreatedStatusEventBody struct {
	AssetData
}

type DeletedStatusEventBody struct {
}

type MovedStatusEventBody struct {
	OldSlot int16 `json:"oldSlot"`
}

type QuantityChangedEventBody struct {
	Quantity uint32 `json:"quantity"`
}
