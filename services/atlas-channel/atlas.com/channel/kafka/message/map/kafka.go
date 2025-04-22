package _map

const (
	EnvEventTopicMapStatus                = "EVENT_TOPIC_MAP_STATUS"
	EventTopicMapStatusTypeCharacterEnter = "CHARACTER_ENTER"
	EventTopicMapStatusTypeCharacterExit  = "CHARACTER_EXIT"
)

type StatusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type CharacterEnter struct {
	CharacterId uint32 `json:"characterId"`
}

type CharacterExit struct {
	CharacterId uint32 `json:"characterId"`
}
