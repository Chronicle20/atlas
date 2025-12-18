package character

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventStatusTypeLogout  = "LOGOUT"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type LogoutStatusEventBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type DeletedStatusEventBody struct {
}
