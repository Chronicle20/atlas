package character

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER"
	CommandRequestChangeMeso = "REQUEST_CHANGE_MESO"
	CommandChangeMap         = "CHANGE_MAP"
)

type CommandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type ChangeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}

type RequestChangeMesoBody struct {
	Amount int32 `json:"amount"`
}

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeMesoChanged    = "MESO_CHANGED"
	StatusEventTypeError          = "ERROR"

	StatusEventErrorTypeNotEnoughMeso = "NOT_ENOUGH_MESO"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

type StatusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type StatusEventChannelChangedBody struct {
	ChannelId    byte   `json:"channelId"`
	OldChannelId byte   `json:"oldChannelId"`
	MapId        uint32 `json:"mapId"`
}

type StatusEventErrorBody[F any] struct {
	Error string `json:"error"`
	Body  F      `json:"body"`
}

type MesoChangedStatusEventBody struct {
	Amount int32 `json:"amount"`
}

type NotEnoughMesoErrorStatusBody struct {
	Amount int32 `json:"amount"`
}
