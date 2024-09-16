package inventory

import (
	"github.com/Chronicle20/atlas-tenant"
)

const (
	EnvCommandTopicEquipItem   = "COMMAND_TOPIC_EQUIP_ITEM"
	EnvCommandTopicUnequipItem = "COMMAND_TOPIC_UNEQUIP_ITEM"
	EnvCommandTopicMoveItem    = "COMMAND_TOPIC_MOVE_ITEM"
	EnvCommandTopicDropItem    = "COMMAND_TOPIC_DROP_ITEM"
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

type moveItemCommand struct {
	Tenant        tenant.Model `json:"tenant"`
	CharacterId   uint32       `json:"characterId"`
	InventoryType byte         `json:"inventoryType"`
	Source        int16        `json:"source"`
	Destination   int16        `json:"destination"`
}

type dropItemCommand struct {
	Tenant        tenant.Model `json:"tenant"`
	CharacterId   uint32       `json:"characterId"`
	InventoryType byte         `json:"inventoryType"`
	Source        int16        `json:"source"`
	Quantity      int16        `json:"quantity"`
}
