package character

import (
	"atlas-character-factory/tenant"
)

const (
	EnvEventTopicCharacterCreated = "EVENT_TOPIC_CHARACTER_CREATED"
	EnvEventTopicItemGain         = "EVENT_TOPIC_ITEM_GAIN"
)

type createdEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	WorldId     byte         `json:"worldId"`
	Name        string       `json:"name"`
}

type gainItemEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	ItemId      uint32       `json:"itemId"`
	Quantity    uint32       `json:"quantity"`
}
