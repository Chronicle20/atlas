package character

const (
	EnvEventTopicStatus           = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeDeleted        = "DELETED"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreatedStatusEventBody struct {
	Name string `json:"name"`
}

type DeletedStatusEventBody struct {
}

type LoginStatusEventBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type LogoutStatusEventBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type ChannelChangedStatusEventBody struct {
	ChannelId    byte   `json:"channelId"`
	OldChannelId byte   `json:"oldChannelId"`
	MapId        uint32 `json:"mapId"`
}
