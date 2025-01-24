package character

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeMesoChanged    = "MESO_CHANGED"
	StatusEventTypeError          = "ERROR"

	StatusEventErrorTypeNotEnoughMeso = "NOT_ENOUGH_MESO"
)

type statusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

type statusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventChannelChangedBody struct {
	ChannelId    byte   `json:"channelId"`
	OldChannelId byte   `json:"oldChannelId"`
	MapId        uint32 `json:"mapId"`
}

type statusEventErrorBody[F any] struct {
	Error string `json:"error"`
	Body  F      `json:"body"`
}

type mesoChangedStatusEventBody struct {
	Amount int32 `json:"amount"`
}

type notEnoughMesoErrorStatusBody struct {
	Amount int32 `json:"amount"`
}
