package drop

const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeSpawnFromCharacter = "SPAWN_FROM_CHARACTER"
)

type command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type spawnFromCharacterCommandBody struct {
	ItemId      uint32 `json:"itemId"`
	EquipmentId uint32 `json:"equipmentId"`
	Quantity    uint32 `json:"quantity"`
	Mesos       uint32 `json:"mesos"`
	DropType    byte   `json:"dropType"`
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	OwnerId     uint32 `json:"ownerId"`
	DropperId   uint32 `json:"dropperId"`
	DropperX    int16  `json:"dropperX"`
	DropperY    int16  `json:"dropperY"`
	PlayerDrop  bool   `json:"playerDrop"`
}
