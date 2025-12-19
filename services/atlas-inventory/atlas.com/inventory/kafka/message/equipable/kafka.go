package equipable

import "time"

const (
	EnvStatusEventTopic = "EVENT_TOPIC_EQUIPABLE_STATUS"
	StatusEventCreated  = "CREATED"
	StatusEventUpdated  = "UPDATED"
	StatusEventDeleted  = "DELETED"
)

type StatusEvent[E any] struct {
	Id   uint32 `json:"id"`
	Type string `json:"type"`
	Body E      `json:"body"`
}

type AttributeBody struct {
	Strength       uint16    `json:"strength"`
	Dexterity      uint16    `json:"dexterity"`
	Intelligence   uint16    `json:"intelligence"`
	Luck           uint16    `json:"luck"`
	HP             uint16    `json:"hp"`
	MP             uint16    `json:"mp"`
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
	OwnerName      string    `json:"ownerName"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      uint8     `json:"levelType"`
	Level          uint8     `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}

type DeletedStatusEventBody struct {
}
