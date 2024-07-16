package character

import (
	"atlas-character-factory/tenant"
)

const (
	EnvEventTopicCharacterStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated = "CREATED"

	EnvEventTopicItemGain     = "EVENT_TOPIC_ITEM_GAIN"
	EnvEventTopicEquipChanged = "EVENT_TOPIC_EQUIP_CHANGED"
)

type statusEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	WorldId     byte         `json:"worldId"`
	Body        E            `json:"body"`
}

type statusEventCreatedBody struct {
	Name string `json:"name"`
}

type gainItemEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	ItemId      uint32       `json:"itemId"`
	Quantity    uint32       `json:"quantity"`
	Slot        int16        `json:"slot"`
}

type equipChangedEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Change      string       `json:"change"`
	ItemId      uint32       `json:"itemId"`
}
