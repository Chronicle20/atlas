package drop

const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeRequestReservation = "REQUEST_RESERVATION"
)

type command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type requestReservationCommandBody struct {
	DropId      uint32 `json:"dropId"`
	CharacterId uint32 `json:"characterId"`
	CharacterX  int16  `json:"characterX"`
	CharacterY  int16  `json:"characterY"`
	PetSlot     int8   `json:"petSlot"`
}
