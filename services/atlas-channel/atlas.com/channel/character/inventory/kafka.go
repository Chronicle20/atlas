package inventory

const (
	EnvCommandTopicEquipItem   = "COMMAND_TOPIC_EQUIP_ITEM"
	EnvCommandTopicUnequipItem = "COMMAND_TOPIC_UNEQUIP_ITEM"
	EnvCommandTopicMoveItem    = "COMMAND_TOPIC_MOVE_ITEM"
	EnvCommandTopicDropItem    = "COMMAND_TOPIC_DROP_ITEM"
)

type equipItemCommand struct {
	CharacterId uint32 `json:"characterId"`
	Source      int16  `json:"source"`
	Destination int16  `json:"destination"`
}

type unequipItemCommand struct {
	CharacterId uint32 `json:"characterId"`
	Source      int16  `json:"source"`
	Destination int16  `json:"destination"`
}

type moveItemCommand struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Source        int16  `json:"source"`
	Destination   int16  `json:"destination"`
}

type dropItemCommand struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Source        int16  `json:"source"`
	Quantity      int16  `json:"quantity"`
}
