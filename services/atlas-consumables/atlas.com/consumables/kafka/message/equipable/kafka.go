package equipable

import "time"

const (
	EnvCommandTopic = "COMMAND_TOPIC_EQUIPABLE"
	CommandChange   = "CHANGE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Id          uint32 `json:"id"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type ChangeBody struct {
	Strength       int16     `json:"strength"`
	Dexterity      int16     `json:"dexterity"`
	Intelligence   int16     `json:"intelligence"`
	Luck           int16     `json:"luck"`
	HP             int16     `json:"hp"`
	MP             int16     `json:"mp"`
	WeaponAttack   int16     `json:"weaponAttack"`
	MagicAttack    int16     `json:"magicAttack"`
	WeaponDefense  int16     `json:"weaponDefense"`
	MagicDefense   int16     `json:"magicDefense"`
	Accuracy       int16     `json:"accuracy"`
	Avoidability   int16     `json:"avoidability"`
	Hands          int16     `json:"hands"`
	Speed          int16     `json:"speed"`
	Jump           int16     `json:"jump"`
	Slots          int16     `json:"slots"`
	OwnerName      string    `json:"ownerName"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      int8      `json:"levelType"`
	Level          int8      `json:"level"`
	Experience     int32     `json:"experience"`
	HammersApplied int32     `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}
