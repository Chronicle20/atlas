package chair

const (
	TypeFixed    = "FIXED"
	TypePortable = "PORTABLE"
)

const (
	EnvCommandTopic    = "COMMAND_TOPIC_CHAIR"
	CommandUseChair    = "USE"
	CommandCancelChair = "CANCEL"
)

type Command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type UseChairCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	ChairType   string `json:"chairType"`
	ChairId     uint32 `json:"chairId"`
}

type CancelChairCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

const (
	EnvEventTopicStatus      = "EVENT_TOPIC_CHAIR_STATUS"
	EventStatusTypeUsed      = "USED"
	EventStatusTypeError     = "ERROR"
	EventStatusTypeCancelled = "CANCELLED"

	ErrorTypeInternal = "INTERNAL"
)

type StatusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	ChairType string `json:"chairType"`
	ChairId   uint32 `json:"chairId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type StatusEventUsedBody struct {
	CharacterId uint32 `json:"characterId"`
}

type StatusEventErrorBody struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
}

type StatusEventCancelledBody struct {
	CharacterId uint32 `json:"characterId"`
}
