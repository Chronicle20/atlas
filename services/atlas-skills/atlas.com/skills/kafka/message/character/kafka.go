package character

const (
	EnvEventTopicCharacterStatus   = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogout = "LOGOUT"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}
