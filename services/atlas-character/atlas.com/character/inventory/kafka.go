package inventory

import (
	"atlas-character/tenant"
)

const (
	EnvCommandTopicEquipItem   = "COMMAND_TOPIC_EQUIP_ITEM"
	EnvCommandTopicUnequipItem = "COMMAND_TOPIC_UNEQUIP_ITEM"
	EnvEventTopicItemGain      = "EVENT_TOPIC_ITEM_GAIN"
	EnvEventTopicEquipChanged  = "EVENT_TOPIC_EQUIP_CHANGED"
)

type equipItemCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Source      int16        `json:"source"`
	Destination int16        `json:"destination"`
}

type unequipItemCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Source      int16        `json:"source"`
}

type gainItemEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	ItemId      uint32       `json:"itemId"`
	Quantity    uint32       `json:"quantity"`
}

type equipChangedEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Change      string       `json:"change"`
	ItemId      uint32       `json:"itemId"`
}
