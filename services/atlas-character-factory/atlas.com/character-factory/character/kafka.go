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

type statusEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Name        string       `json:"name"`
	WorldId     byte         `json:"worldId"`
	ChannelId   byte         `json:"channelId"`
	MapId       uint32       `json:"mapId"`
	Type        string       `json:"type"`
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
