package character

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeMapChanged     = "MAP_CHANGED"
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

type StatusEventMapChangedBody struct {
	ChannelId      byte   `json:"channelId"`
	OldMapId       uint32 `json:"oldMapId"`
	TargetMapId    uint32 `json:"targetMapId"`
	TargetPortalId uint32 `json:"targetPortalId"`
}

type ChangeChannelEventLoginBody struct {
	ChannelId    byte   `json:"channelId"`
	OldChannelId byte   `json:"oldChannelId"`
	MapId        uint32 `json:"mapId"`
}
