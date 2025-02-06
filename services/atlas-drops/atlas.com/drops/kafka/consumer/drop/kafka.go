package drop

const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeSpawn              = "SPAWN"
	CommandTypeSpawnFromCharacter = "SPAWN_FROM_CHARACTER"
	CommandTypeRequestReservation = "REQUEST_RESERVATION"
	CommandTypeCancelReservation  = "CANCEL_RESERVATION"
	CommandTypeRequestPickUp      = "REQUEST_PICK_UP"
)

type command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type spawnCommandBody struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Mesos        uint32 `json:"mesos"`
	DropType     byte   `json:"dropType"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	OwnerId      uint32 `json:"ownerId"`
	OwnerPartyId uint32 `json:"ownerPartyId"`
	DropperId    uint32 `json:"dropperId"`
	DropperX     int16  `json:"dropperX"`
	DropperY     int16  `json:"dropperY"`
	PlayerDrop   bool   `json:"playerDrop"`
	Mod          byte   `json:"mod"`
}

type spawnFromCharacterCommandBody struct {
	ItemId       uint32 `json:"itemId"`
	EquipmentId  uint32 `json:"equipmentId"`
	Quantity     uint32 `json:"quantity"`
	Mesos        uint32 `json:"mesos"`
	DropType     byte   `json:"dropType"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	OwnerId      uint32 `json:"ownerId"`
	OwnerPartyId uint32 `json:"ownerPartyId"`
	DropperId    uint32 `json:"dropperId"`
	DropperX     int16  `json:"dropperX"`
	DropperY     int16  `json:"dropperY"`
	PlayerDrop   bool   `json:"playerDrop"`
	Mod          byte   `json:"mod"`
}

type requestReservationCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
	CharacterX  int16  `json:"characterX"`
	CharacterY  int16  `json:"characterY"`
}

type cancelReservationCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
}

type requestPickUpCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
}
