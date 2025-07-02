package character

const (
	EnvEventTopicCharacterStatus           = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeLogin          = "LOGIN"
	EventCharacterStatusTypeLogout         = "LOGOUT"
	EventCharacterStatusTypeChannelChanged = "CHANNEL_CHANGED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventLoginBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
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
