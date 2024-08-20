package inventory

import (
	"atlas-channel/tenant"
)

const (
	EnvCommandTopicEquipItem   = "COMMAND_TOPIC_EQUIP_ITEM"
	EnvCommandTopicUnequipItem = "COMMAND_TOPIC_UNEQUIP_ITEM"
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
	Destination int16        `json:"destination"`
}
