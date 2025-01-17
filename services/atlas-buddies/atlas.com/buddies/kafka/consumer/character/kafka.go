package character

const (
	EnvEventTopicCharacterStatus   = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin  = "LOGIN"
	EventCharacterStatusTypeLogout = "LOGOUT"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type statusEventLoginBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}
