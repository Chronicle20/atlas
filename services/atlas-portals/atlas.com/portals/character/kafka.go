package character

import "atlas-portals/tenant"

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

type statusEvent[E any] struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Type        string       `json:"type"`
	WorldId     byte         `json:"worldId"`
	Body        E            `json:"body"`
}

// TODO this should transmit stats
type statusEventStatChangedBody struct {
	ChannelId       byte `json:"channelId"`
	ExclRequestSent bool `json:"exclRequestSent"`
}
