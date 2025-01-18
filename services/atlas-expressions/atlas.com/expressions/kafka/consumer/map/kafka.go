package _map

const (
	EnvEventTopicMapStatus               = "EVENT_TOPIC_MAP_STATUS"
	EventTopicMapStatusTypeCharacterExit = "CHARACTER_EXIT"
)

type statusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type characterExit struct {
	CharacterId uint32 `json:"characterId"`
}
